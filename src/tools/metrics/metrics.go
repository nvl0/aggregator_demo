// Package metrics prometheus-метрики агрегатора на собственном реестре.
package metrics

import (
	"database/sql"
	"log/slog"
	"net/http"

	"aggregator/src/tools/logger"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// defaultVersion версия сборки, если VERSION не задан
	defaultVersion = "dev"
	// dbStatsName имя бд в сериях go_sql_*
	dbStatsName = "aggregator"
	// buildInfoValue значение серии aggregator_build_info, всегда 1
	buildInfoValue = 1

	// labelVersion лейбл версии сборки
	labelVersion = "version"
	// labelReason лейбл причины раннего выхода цикла
	labelReason = "reason"
	// labelPhase лейбл фазы
	labelPhase = "phase"
	// labelResult лейбл исхода обработки nas_ip
	labelResult = "result"
	// labelStage лейбл этапа, на котором произошла ошибка
	labelStage = "stage"
	// labelDirection лейбл направления трафика
	labelDirection = "direction"

	// flowSizeBucketStart нижняя граница гистограммы размера flow, байт
	flowSizeBucketStart = 1024
	// flowSizeBucketFactor множитель шага гистограммы размера flow
	flowSizeBucketFactor = 4
	// flowSizeBucketCount количество границ гистограммы размера flow
	flowSizeBucketCount = 8
)

var (
	// cycleDurationBuckets границы длительности цикла, секунды.
	// верхняя 120 при StartDur 60s показывает переработку
	cycleDurationBuckets = []float64{1, 2, 5, 10, 20, 30, 45, 60, 90, 120}

	// phaseDurationBuckets границы длительности фаз, секунды
	phaseDurationBuckets = []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30}

	// flowSizeBuckets границы размера flow, байты: 1 KiB ... ~16 MiB
	flowSizeBuckets = prometheus.ExponentialBuckets(
		flowSizeBucketStart, flowSizeBucketFactor, flowSizeBucketCount)
)

// Metrics набор метрик агрегатора на собственном реестре.
// Собственный реестр вместо глобального: регистрация изолирована в тестах,
// не течет между ними и не тянет метрики чужих библиотек
type Metrics struct {
	log *slog.Logger
	reg *prometheus.Registry

	buildInfo *prometheus.GaugeVec

	cycleDuration     prometheus.Histogram
	lastCycleDuration prometheus.Gauge
	cycleInProgress   prometheus.Gauge
	lastSuccess       prometheus.Gauge
	cycleErrors       *prometheus.CounterVec
	nasDiscovered     prometheus.Gauge
	phaseDuration     *prometheus.HistogramVec

	nasProcessed     *prometheus.CounterVec
	nasErrors        *prometheus.CounterVec
	nasPhaseDuration *prometheus.HistogramVec
	chunksSaved      prometheus.Counter
	flowSize         prometheus.Histogram
	accountedTraffic *prometheus.CounterVec

	workerPanics prometheus.Counter
	poolSize     prometheus.Gauge
}

// New собирает метрики на новом реестре.
// db может быть nil: тогда серии go_sql_* не публикуются
func New(log *slog.Logger, version string, db *sql.DB) *Metrics {
	if version == "" {
		version = defaultVersion
	}

	m := &Metrics{
		log: log,
		reg: prometheus.NewRegistry(),
	}

	collectorList := make([]prometheus.Collector, 0)
	collectorList = append(collectorList, m.initCycleMetrics()...)
	collectorList = append(collectorList, m.initNASMetrics()...)
	collectorList = append(collectorList, m.initStaticMetrics()...)
	collectorList = append(collectorList,
		collectors.NewGoCollector(),
		collectors.NewBuildInfoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	if db != nil {
		collectorList = append(collectorList, collectors.NewDBStatsCollector(db, dbStatsName))
	}

	m.register(collectorList...)

	m.buildInfo.WithLabelValues(version).Set(buildInfoValue)
	m.preinit()

	return m
}

// Nop метрики в выброшенный реестр: используются как дефолт там,
// где реальный экземпляр не передан (тесты, loadgen)
func Nop() *Metrics {
	return New(logger.NewDiscard(), "", nil)
}

// Handler http-хендлер экспозиции метрик.
// ErrorLog намеренно не выставляем: это потребовало бы импорта log,
// запрещенного depguard вне main.go
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{})
}

// register регистрирует коллекторы, не роняя процесс на ошибке регистрации
func (m *Metrics) register(collectorList ...prometheus.Collector) {
	for _, c := range collectorList {
		if err := m.reg.Register(c); err != nil {
			m.log.Error("не удалось зарегистрировать коллектор метрик, ошибка", "error", err)
		}
	}
}

// initCycleMetrics создает метрики уровня цикла
func (m *Metrics) initCycleMetrics() []prometheus.Collector {
	m.cycleDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "aggregator_cycle_duration_seconds",
		Help:    "длительность полного цикла агрегации в секундах",
		Buckets: cycleDurationBuckets,
	})
	m.lastCycleDuration = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "aggregator_last_cycle_duration_seconds",
		Help: "длительность последнего завершившегося цикла агрегации в секундах",
	})
	m.cycleInProgress = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "aggregator_cycle_in_progress",
		Help: "признак идущего цикла агрегации, 1 идет и 0 не идет",
	})
	m.lastSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "aggregator_last_success_timestamp_seconds",
		Help: "unix-время последнего цикла, дошедшего до рассылки nas_ip",
	})
	m.cycleErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "aggregator_cycle_errors_total",
		Help: "количество ранних выходов цикла агрегации по причинам",
	}, []string{labelReason})
	m.nasDiscovered = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "aggregator_nas_discovered",
		Help: "количество nas_ip директорий, найденных в последнем цикле",
	})
	m.phaseDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "aggregator_phase_duration_seconds",
		Help:    "длительность фаз цикла агрегации в секундах",
		Buckets: phaseDurationBuckets,
	}, []string{labelPhase})

	return []prometheus.Collector{
		m.cycleDuration, m.lastCycleDuration, m.cycleInProgress,
		m.lastSuccess, m.cycleErrors, m.nasDiscovered, m.phaseDuration,
	}
}

// initNASMetrics создает метрики уровня nas_ip
func (m *Metrics) initNASMetrics() []prometheus.Collector {
	m.nasProcessed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "aggregator_nas_processed_total",
		Help: "количество обработанных nas_ip по исходам",
	}, []string{labelResult})
	m.nasErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "aggregator_nas_errors_total",
		Help: "количество ошибок обработки nas_ip по этапам",
	}, []string{labelStage})
	m.nasPhaseDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "aggregator_nas_phase_duration_seconds",
		Help:    "длительность фаз обработки одного nas_ip в секундах",
		Buckets: phaseDurationBuckets,
	}, []string{labelPhase})
	m.chunksSaved = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "aggregator_chunks_saved_total",
		Help: "количество чанков трафика, сохраненных в бд",
	})
	m.flowSize = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "aggregator_flow_size_bytes",
		Help:    "размер собранного flow одного nas_ip в байтах",
		Buckets: flowSizeBuckets,
	})
	m.accountedTraffic = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "aggregator_accounted_traffic_bytes_total",
		Help: "объем учтенного трафика в байтах по направлениям",
	}, []string{labelDirection})

	return []prometheus.Collector{
		m.nasProcessed, m.nasErrors, m.nasPhaseDuration,
		m.chunksSaved, m.flowSize, m.accountedTraffic,
	}
}

// initStaticMetrics создает метрики воркеров и статики
func (m *Metrics) initStaticMetrics() []prometheus.Collector {
	m.workerPanics = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "aggregator_worker_panics_total",
		Help: "количество паник воркеров агрегации",
	})
	m.poolSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "aggregator_pool_size",
		Help: "размер пула воркеров агрегации",
	})
	m.buildInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "aggregator_build_info",
		Help: "версия сборки агрегатора, значение всегда 1",
	}, []string{labelVersion})

	return []prometheus.Collector{m.workerPanics, m.poolSize, m.buildInfo}
}

// preinit создает все серии лейблованных векторов с нулями,
// чтобы rate() и алерты по редким исходам работали с t=0
func (m *Metrics) preinit() {
	for _, reason := range allCycleErrReasons {
		m.cycleErrors.WithLabelValues(string(reason))
	}

	for _, phase := range allPhases {
		m.phaseDuration.WithLabelValues(string(phase))
	}

	for _, result := range allNASResults {
		m.nasProcessed.WithLabelValues(string(result))
	}

	for _, stage := range allNASStages {
		m.nasErrors.WithLabelValues(string(stage))
	}

	for _, phase := range allNASPhases {
		m.nasPhaseDuration.WithLabelValues(string(phase))
	}

	for _, direction := range allDirections {
		m.accountedTraffic.WithLabelValues(string(direction))
	}
}
