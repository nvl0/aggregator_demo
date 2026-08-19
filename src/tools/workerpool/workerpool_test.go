package workerpool_test

import (
	"aggregator/src/tools/workerpool"
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestGoLimit количество одновременно работающих горутин не превышает размер пула
func TestGoLimit(t *testing.T) {
	assertMaxPeak := func(wantMax int32) func(t *testing.T, peak int32) {
		return func(t *testing.T, peak int32) {
			if peak > wantMax {
				t.Fatalf("одновременно работало %d горутин, ожидалось не больше %d", peak, wantMax)
			}
		}
	}

	tests := []struct {
		name       string
		poolSize   int
		taskCount  int
		assertFunc func(t *testing.T, peak int32)
	}{
		{name: "пул меньше числа задач", poolSize: 3, taskCount: 50, assertFunc: assertMaxPeak(3)},
		{name: "пул больше числа задач", poolSize: 10, taskCount: 4, assertFunc: assertMaxPeak(4)},
		{name: "нулевой размер приводится к единице", poolSize: 0, taskCount: 10, assertFunc: assertMaxPeak(1)},
		{name: "отрицательный размер приводится к единице", poolSize: -5, taskCount: 10, assertFunc: assertMaxPeak(1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var current, peak atomic.Int32

			p := workerpool.New(tt.poolSize)

			for i := 0; i < tt.taskCount; i++ {
				if ok := p.Go(context.Background(), func() {
					now := current.Add(1)

					for {
						old := peak.Load()
						if now <= old || peak.CompareAndSwap(old, now) {
							break
						}
					}

					time.Sleep(time.Millisecond)
					current.Add(-1)
				}); !ok {
					t.Fatal("Go вернул false при неотмененном контексте")
				}
			}

			p.Wait()

			tt.assertFunc(t, peak.Load())

			if got := current.Load(); got != 0 {
				t.Fatalf("после Wait осталось %d незавершенных горутин", got)
			}
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
	if ok := p.Go(context.Background(), func() {
		close(started)
		<-release
	}); !ok {
		t.Fatal("Go вернул false при неотмененном контексте")
	}

	<-started

	ctx, cancel := context.WithCancel(context.Background())
	var executed atomic.Bool

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	if ok := p.Go(ctx, func() { executed.Store(true) }); ok {
		t.Fatal("Go вернул true, хотя контекст был отменен во время ожидания слота")
	}

	close(release)
	p.Wait()

	if executed.Load() {
		t.Fatal("fn была запущена несмотря на отмену контекста")
	}
}

// TestGoCtxAlreadyCanceled заранее отмененный контекст не запускает fn даже при свободном слоте
func TestGoCtxAlreadyCanceled(t *testing.T) {
	p := workerpool.New(10)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var executed atomic.Bool

	if ok := p.Go(ctx, func() { executed.Store(true) }); ok {
		t.Fatal("Go вернул true при заранее отмененном контексте")
	}

	p.Wait()

	if executed.Load() {
		t.Fatal("fn была запущена при заранее отмененном контексте")
	}
}

// TestWait дожидается завершения всех уже запущенных горутин
func TestWait(t *testing.T) {
	const taskCount = 20

	var done atomic.Int32

	p := workerpool.New(4)

	for i := 0; i < taskCount; i++ {
		if ok := p.Go(context.Background(), func() {
			time.Sleep(5 * time.Millisecond)
			done.Add(1)
		}); !ok {
			t.Fatal("Go вернул false при неотмененном контексте")
		}
	}

	p.Wait()

	if got := done.Load(); got != taskCount {
		t.Fatalf("завершилось %d горутин, ожидалось %d", got, taskCount)
	}
}
