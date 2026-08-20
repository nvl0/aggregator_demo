package workerpool_test

import (
	"aggregator/src/tools/workerpool"
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoLimit количество одновременно работающих горутин не превышает размер пула
func TestGoLimit(t *testing.T) {
	tests := []struct {
		name       string
		poolSize   int
		taskCount  int
		assertFunc func(t *testing.T, peak, poolSize int32)
	}{
		{
			name:      "пул меньше числа задач",
			poolSize:  3,
			taskCount: 50,
			assertFunc: func(t *testing.T, peak, poolSize int32) {
				assert.LessOrEqual(
					t,
					peak,
					poolSize,
					"одновременно работало %d горутин, ожидалось не больше %d",
					peak,
					poolSize,
				)
			}},
		{
			name:      "пул больше числа задач",
			poolSize:  10,
			taskCount: 4,
			assertFunc: func(t *testing.T, peak, poolSize int32) {
				assert.LessOrEqual(
					t,
					peak,
					poolSize,
					"одновременно работало %d горутин, ожидалось не больше %d",
					peak,
					poolSize,
				)
			},
		},
		{
			name:      "нулевой размер приводится к единице",
			poolSize:  0,
			taskCount: 10,
			assertFunc: func(t *testing.T, peak, poolSize int32) {
				assert.LessOrEqual(
					t,
					peak,
					poolSize,
					"одновременно работало %d горутин, ожидалось не больше %d",
					peak,
					poolSize,
				)
			},
		},
		{
			name:      "отрицательный размер приводится к единице",
			poolSize:  -5,
			taskCount: 10,
			assertFunc: func(t *testing.T, peak, poolSize int32) {
				assert.LessOrEqual(
					t,
					peak,
					poolSize,
					"одновременно работало %d горутин, ожидалось не больше %d",
					peak,
					poolSize,
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var current, peak atomic.Int32

			p := workerpool.New(tt.poolSize)

			for range tt.taskCount {
				ok := p.Go(context.Background(), func() {
					now := current.Add(1)

					for {
						old := peak.Load()
						if now <= old || peak.CompareAndSwap(old, now) {
							break
						}
					}

					time.Sleep(time.Millisecond)
					current.Add(-1)
				})
				require.True(t, ok, "Go вернул false при неотмененном контексте")
			}

			p.Wait()
			require.Zero(t, current.Load(), "после Wait осталось %d незавершенных горутин", current.Load())

			wantMax := max(int32(tt.poolSize), 1)
			tt.assertFunc(t, peak.Load(), wantMax)
		})
	}
}

// TestGoCtxCancel Go возвращает false и не запускает fn,
// если контекст отменился во время ожидания свободного слота
func TestGoCtxCancel(t *testing.T) {
	p := workerpool.New(1)

	started := make(chan struct{})
	release := make(chan struct{})

	// занимаем единственный слот пула
	ok := p.Go(context.Background(), func() {
		close(started)
		<-release
	})
	require.True(t, ok, "Go вернул false при неотмененном контексте")

	<-started

	ctx, cancel := context.WithCancel(context.Background())
	var executed atomic.Bool

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	ok = p.Go(ctx, func() { executed.Store(true) })
	require.False(t, ok, "Go вернул true, хотя контекст был отменен во время ожидания слота")

	close(release)
	p.Wait()

	require.False(t, executed.Load(), "fn была запущена несмотря на отмену контекста")
}

// TestGoCtxAlreadyCanceled заранее отмененный контекст не запускает fn даже при свободном слоте
func TestGoCtxAlreadyCanceled(t *testing.T) {
	p := workerpool.New(10)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var executed atomic.Bool

	ok := p.Go(ctx, func() { executed.Store(true) })
	require.False(t, ok, "Go вернул true при заранее отмененном контексте")

	p.Wait()

	require.False(t, executed.Load(), "fn была запущена при заранее отмененном контексте")
}

// TestWait дожидается завершения всех уже запущенных горутин
func TestWait(t *testing.T) {
	const taskCount = 20

	var done atomic.Int32

	p := workerpool.New(4)

	for range taskCount {
		ok := p.Go(context.Background(), func() {
			time.Sleep(5 * time.Millisecond)
			done.Add(1)
		})
		require.True(t, ok, "Go вернул false при неотмененном контексте")
	}

	p.Wait()
	require.EqualValues(t, taskCount, done.Load(), "завершилось не столько горутин, сколько ожидалось")
}

// TestGoRecover паника в одной задаче не роняет пул, не выходит наружу
// за пределы Go/Wait и не приводит к утечке слота семафора
func TestGoRecover(t *testing.T) {
	const (
		taskCount  = 10
		panicValue = "паника тестовой задачи"
	)

	tests := []struct {
		name        string
		withOnPanic bool
		assertFunc  func(t *testing.T, recoveredList []any)
	}{
		{
			name:        "колбэк получает восстановленное значение",
			withOnPanic: true,
			assertFunc: func(t *testing.T, recoveredList []any) {
				require.Len(t, recoveredList, 1, "onPanic вызван не один раз")
				assert.Equal(
					t,
					panicValue,
					recoveredList[0],
					"onPanic получил не то значение, которое было передано в panic",
				)
			},
		},
		{
			name:        "без колбэка паника гасится молча",
			withOnPanic: false,
			assertFunc: func(t *testing.T, recoveredList []any) {
				assert.Empty(t, recoveredList, "onPanic не задан, но что-то было записано")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				mu            sync.Mutex
				recoveredList []any
				done          atomic.Int32
			)

			optList := make([]workerpool.Option, 0, 1)
			if tt.withOnPanic {
				optList = append(optList, workerpool.WithOnPanic(func(recovered any) {
					mu.Lock()
					defer mu.Unlock()

					recoveredList = append(recoveredList, recovered)
				}))
			}

			// размер пула 1: если паника не освободит слот семафора,
			// следующий Go заблокируется навсегда и тест упадет по таймауту
			p := workerpool.New(1, optList...)

			ok := p.Go(context.Background(), func() { panic(panicValue) })
			require.True(t, ok, "Go вернул false при неотмененном контексте")

			for range taskCount {
				ok = p.Go(context.Background(), func() { done.Add(1) })
				require.True(t, ok, "Go вернул false после паники в другой задаче")
			}

			p.Wait()
			require.EqualValues(t, taskCount, done.Load(),
				"после паники выполнились не все оставшиеся задачи")

			mu.Lock()
			defer mu.Unlock()

			tt.assertFunc(t, recoveredList)
		})
	}
}
