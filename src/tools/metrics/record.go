package metrics

import "time"

const (
	// inProgressOn значение метрики идущего цикла
	inProgressOn = 1
	// inProgressOff значение метрики завершенного цикла
	inProgressOff = 0
)

// ObserveCycle фиксирует длительность завершившегося цикла
func (m *Metrics) ObserveCycle(d time.Duration) {
	if m == nil {
		return
	}

	m.cycleDuration.Observe(d.Seconds())
}

// SetLastCycleDuration выставляет длительность последнего цикла
func (m *Metrics) SetLastCycleDuration(d time.Duration) {
	if m == nil {
		return
	}

	m.lastCycleDuration.Set(d.Seconds())
}

// SetCycleInProgress выставляет признак идущего цикла
func (m *Metrics) SetCycleInProgress(inProgress bool) {
	if m == nil {
		return
	}

	value := float64(inProgressOff)
	if inProgress {
		value = inProgressOn
	}

	m.cycleInProgress.Set(value)
}

// SetLastSuccess отмечает текущее время как время последнего успешного цикла
func (m *Metrics) SetLastSuccess() {
	if m == nil {
		return
	}

	m.lastSuccess.Set(float64(time.Now().Unix()))
}

// IncCycleError считает ранний выход цикла по причине
func (m *Metrics) IncCycleError(reason CycleErrReason) {
	if m == nil {
		return
	}

	m.cycleErrors.WithLabelValues(string(reason)).Inc()
}

// SetNASDiscovered выставляет количество найденных nas_ip директорий
func (m *Metrics) SetNASDiscovered(n int) {
	if m == nil {
		return
	}

	m.nasDiscovered.Set(float64(n))
}

// ObservePhase фиксирует длительность фазы цикла
func (m *Metrics) ObservePhase(p Phase, d time.Duration) {
	if m == nil {
		return
	}

	m.phaseDuration.WithLabelValues(string(p)).Observe(d.Seconds())
}

// IncNAS считает исход обработки одного nas_ip
func (m *Metrics) IncNAS(r NASResult) {
	if m == nil {
		return
	}

	m.nasProcessed.WithLabelValues(string(r)).Inc()
}

// IncNASError считает ошибку обработки nas_ip по этапу
func (m *Metrics) IncNASError(s NASStage) {
	if m == nil {
		return
	}

	m.nasErrors.WithLabelValues(string(s)).Inc()
}

// ObserveNASPhase фиксирует длительность фазы обработки nas_ip
func (m *Metrics) ObserveNASPhase(p NASPhase, d time.Duration) {
	if m == nil {
		return
	}

	m.nasPhaseDuration.WithLabelValues(string(p)).Observe(d.Seconds())
}

// AddChunksSaved считает сохраненные в бд чанки
func (m *Metrics) AddChunksSaved(n int) {
	if m == nil {
		return
	}

	m.chunksSaved.Add(float64(n))
}

// ObserveFlowSize фиксирует размер собранного flow в байтах
func (m *Metrics) ObserveFlowSize(bytes int) {
	if m == nil {
		return
	}

	m.flowSize.Observe(float64(bytes))
}

// AddAccountedTraffic считает учтенный трафик по направлению
func (m *Metrics) AddAccountedTraffic(dir Direction, bytes int) {
	if m == nil {
		return
	}

	m.accountedTraffic.WithLabelValues(string(dir)).Add(float64(bytes))
}

// IncWorkerPanic считает панику воркера агрегации
func (m *Metrics) IncWorkerPanic() {
	if m == nil {
		return
	}

	m.workerPanics.Inc()
}

// SetPoolSize выставляет размер пула воркеров агрегации
func (m *Metrics) SetPoolSize(n int) {
	if m == nil {
		return
	}

	m.poolSize.Set(float64(n))
}
