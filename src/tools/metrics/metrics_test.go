package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aggregator/src/tools/logger"
	"aggregator/src/tools/metrics"
)

var testLogger = logger.NewNoFileLogger("test")

// scrape снимает текущее состояние метрик через http-хендлер
func scrape(t *testing.T, m *metrics.Metrics) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	m.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("скрейп вернул код %d, ожидался %d", rec.Code, http.StatusOK)
	}

	return rec.Body.String()
}

// requireLine проверяет наличие строки среди снятых серий
func requireLine(t *testing.T, body, want string) {
	t.Helper()

	for _, line := range strings.Split(body, "\n") {
		if strings.TrimSpace(line) == want {
			return
		}
	}

	t.Errorf("в выводе метрик нет строки %q", want)
}

func TestNewExposesBaseSeries(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    []string
	}{
		{
			name:    "версия задана",
			version: "v1.2.3",
			want:    []string{`aggregator_build_info{version="v1.2.3"} 1`},
		},
		{
			name:    "пустая версия подменяется на dev",
			version: "",
			want:    []string{`aggregator_build_info{version="dev"} 1`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := metrics.New(testLogger, tt.version, nil)

			body := scrape(t, m)

			for _, want := range tt.want {
				requireLine(t, body, want)
			}

			// коллекторы рантайма подключены
			if !strings.Contains(body, "go_goroutines") {
				t.Error("в выводе метрик нет серии go_goroutines")
			}
		})
	}
}

func TestNewPreinitializesLabeledSeries(t *testing.T) {
	m := metrics.New(testLogger, "test", nil)

	body := scrape(t, m)

	// все значения enum обязаны присутствовать с нуля,
	// иначе rate() и алерты по редким исходам не работают с t=0
	want := []string{
		`aggregator_cycle_errors_total{reason="dirs_read"} 0`,
		`aggregator_cycle_errors_total{reason="nil_maps"} 0`,
		`aggregator_nas_processed_total{result="ok"} 0`,
		`aggregator_nas_processed_total{result="no_new"} 0`,
		`aggregator_nas_processed_total{result="no_internal"} 0`,
		`aggregator_nas_processed_total{result="unrecognized"} 0`,
		`aggregator_nas_processed_total{result="error"} 0`,
		`aggregator_nas_errors_total{stage="checkpoint"} 0`,
		`aggregator_nas_errors_total{stage="prepare"} 0`,
		`aggregator_nas_errors_total{stage="parse"} 0`,
		`aggregator_nas_errors_total{stage="sift"} 0`,
		`aggregator_nas_errors_total{stage="save"} 0`,
		`aggregator_accounted_traffic_bytes_total{direction="download"} 0`,
		`aggregator_accounted_traffic_bytes_total{direction="upload"} 0`,
	}

	for _, line := range want {
		requireLine(t, body, line)
	}

	// гистограммы с лейблами преинициализированы: считаем счетчики наблюдений
	for _, phase := range []string{"read_dirs", "load_channels", "load_sessions"} {
		requireLine(t, body, `aggregator_phase_duration_seconds_count{phase="`+phase+`"} 0`)
	}

	for _, phase := range []string{"prepare_flow", "parse_flow", "sift_traffic", "save_chunks"} {
		requireLine(t, body, `aggregator_nas_phase_duration_seconds_count{phase="`+phase+`"} 0`)
	}
}

func TestNopIsIsolated(t *testing.T) {
	first := metrics.Nop()
	second := metrics.Nop()

	// два Nop не делят реестр: регистрация второго не паникует и не ломается
	first.IncNAS(metrics.NASResultOK)

	body := scrape(t, second)

	requireLine(t, body, `aggregator_nas_processed_total{result="ok"} 0`)
}

func TestRecordMethods(t *testing.T) {
	tests := []struct {
		name   string
		record func(m *metrics.Metrics)
		want   []string
	}{
		{
			name: "исход обработки nas_ip",
			record: func(m *metrics.Metrics) {
				m.IncNAS(metrics.NASResultOK)
				m.IncNAS(metrics.NASResultOK)
				m.IncNAS(metrics.NASResultNoNew)
			},
			want: []string{
				`aggregator_nas_processed_total{result="ok"} 2`,
				`aggregator_nas_processed_total{result="no_new"} 1`,
				`aggregator_nas_processed_total{result="error"} 0`,
			},
		},
		{
			name: "ошибка этапа обработки nas_ip",
			record: func(m *metrics.Metrics) {
				m.IncNASError(metrics.NASStageParse)
			},
			want: []string{
				`aggregator_nas_errors_total{stage="parse"} 1`,
				`aggregator_nas_errors_total{stage="save"} 0`,
			},
		},
		{
			name: "сохраненные чанки и учтенный трафик",
			record: func(m *metrics.Metrics) {
				m.AddChunksSaved(3)
				m.AddAccountedTraffic(metrics.DirectionDownload, 1000)
				m.AddAccountedTraffic(metrics.DirectionUpload, 250)
			},
			want: []string{
				`aggregator_chunks_saved_total 3`,
				`aggregator_accounted_traffic_bytes_total{direction="download"} 1000`,
				`aggregator_accounted_traffic_bytes_total{direction="upload"} 250`,
			},
		},
		{
			name: "длительность цикла",
			record: func(m *metrics.Metrics) {
				m.ObserveCycle(2 * time.Second)
				m.SetLastCycleDuration(2 * time.Second)
			},
			want: []string{
				`aggregator_cycle_duration_seconds_count 1`,
				`aggregator_cycle_duration_seconds_sum 2`,
				`aggregator_last_cycle_duration_seconds 2`,
			},
		},
		{
			name: "признак идущего цикла",
			record: func(m *metrics.Metrics) {
				m.SetCycleInProgress(true)
			},
			want: []string{`aggregator_cycle_in_progress 1`},
		},
		{
			name: "признак идущего цикла снят",
			record: func(m *metrics.Metrics) {
				m.SetCycleInProgress(true)
				m.SetCycleInProgress(false)
			},
			want: []string{`aggregator_cycle_in_progress 0`},
		},
		{
			name: "причина раннего выхода цикла",
			record: func(m *metrics.Metrics) {
				m.IncCycleError(metrics.CycleErrDirsRead)
			},
			want: []string{
				`aggregator_cycle_errors_total{reason="dirs_read"} 1`,
				`aggregator_cycle_errors_total{reason="nil_maps"} 0`,
			},
		},
		{
			name: "статика цикла",
			record: func(m *metrics.Metrics) {
				m.SetNASDiscovered(7)
				m.SetPoolSize(10)
				m.IncWorkerPanic()
			},
			want: []string{
				`aggregator_nas_discovered 7`,
				`aggregator_pool_size 10`,
				`aggregator_worker_panics_total 1`,
			},
		},
		{
			name: "фазы цикла и nas_ip",
			record: func(m *metrics.Metrics) {
				m.ObservePhase(metrics.PhaseReadDirs, 100*time.Millisecond)
				m.ObserveNASPhase(metrics.NASPhaseParseFlow, 100*time.Millisecond)
			},
			want: []string{
				`aggregator_phase_duration_seconds_count{phase="read_dirs"} 1`,
				`aggregator_nas_phase_duration_seconds_count{phase="parse_flow"} 1`,
			},
		},
		{
			name: "размер flow",
			record: func(m *metrics.Metrics) {
				m.ObserveFlowSize(2048)
			},
			want: []string{
				`aggregator_flow_size_bytes_count 1`,
				`aggregator_flow_size_bytes_sum 2048`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := metrics.New(testLogger, "test", nil)

			tt.record(m)

			body := scrape(t, m)

			for _, want := range tt.want {
				requireLine(t, body, want)
			}
		})
	}
}

func TestNilReceiverIsSafe(_ *testing.T) {
	var m *metrics.Metrics

	// метод на nil-приемнике не должен ронять процесс:
	// защита на случай, если экземпляр не проброшен
	m.IncNAS(metrics.NASResultOK)
	m.IncNASError(metrics.NASStageSave)
	m.IncCycleError(metrics.CycleErrNilMaps)
	m.ObserveCycle(time.Second)
	m.SetLastCycleDuration(time.Second)
	m.SetCycleInProgress(true)
	m.SetLastSuccess()
	m.SetNASDiscovered(1)
	m.ObservePhase(metrics.PhaseReadDirs, time.Second)
	m.ObserveNASPhase(metrics.NASPhaseParseFlow, time.Second)
	m.AddChunksSaved(1)
	m.ObserveFlowSize(1)
	m.AddAccountedTraffic(metrics.DirectionUpload, 1)
	m.IncWorkerPanic()
	m.SetPoolSize(1)
}
