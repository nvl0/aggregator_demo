package config_test

import (
	"aggregator/src/config"
	"testing"
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
