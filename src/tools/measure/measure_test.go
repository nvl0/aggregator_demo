package measure_test

import (
	"sync"
	"testing"

	"aggregator/src/tools/measure"
)

type noopWriter struct{}

func (noopWriter) Write(string) {}

// TestResultRaceWithStartStop проверяет, что Result() можно безопасно вызывать
// параллельно с Start/Stop той же горутиной, что их пишет, без join между ними
// (сценарий: воркер продолжает измерения, пока кто-то читает промежуточный результат).
func TestResultRaceWithStartStop(t *testing.T) {
	t.Setenv("MEASURE", "enable")

	m := measure.NewMeasure(noopWriter{})

	const iterations = 500

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		for range iterations {
			m.Start("op")
			m.Stop("op")
		}
	}()

	go func() {
		defer wg.Done()

		for range iterations {
			m.Result()
		}
	}()

	wg.Wait()
}
