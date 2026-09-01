package config_test

import (
	"testing"

	"aggregator/src/config"
)

// TestWorkerPoolSize размер пула берется из conf.yaml и переопределяется env
func TestWorkerPoolSize(t *testing.T) {
	tests := []struct {
		name   string
		envVal string
		want   int
	}{
		{name: "значение из conf.yaml", envVal: "", want: 10},
		{name: "переопределение через env", envVal: "42", want: 42},
		{name: "нечисловой env игнорируется", envVal: "abc", want: 10},
		{name: "нулевой env игнорируется", envVal: "0", want: 10},
		{name: "отрицательный env игнорируется", envVal: "-3", want: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("WORKER_POOL_SIZE", tt.envVal)

			c, err := config.NewConfig("conf.yaml")
			if err != nil {
				t.Fatalf("не удалось загрузить конфиг, ошибка %v", err)
			}

			if got := c.WorkerPoolSize(); got != tt.want {
				t.Fatalf("WorkerPoolSize() = %d, ожидалось %d", got, tt.want)
			}
		})
	}
}

// TestMetricsAddr адрес служебного http сервера берется из env, иначе дефолт
func TestMetricsAddr(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want string
	}{
		{
			name: "env пуст, берется дефолт",
			env:  "",
			want: ":2112",
		},
		{
			name: "env переопределяет дефолт",
			env:  "127.0.0.1:19999",
			want: "127.0.0.1:19999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("METRICS_ADDR", tt.env)

			c := config.Config{}
			if got := c.MetricsAddr(); got != tt.want {
				t.Errorf("MetricsAddr() = %q, ожидалось %q", got, tt.want)
			}
		})
	}
}
