package config_test

import (
	"testing"

	"aggregator/src/config"
)

// pgEnvKeys переменные окружения, участвующие в строке подключения:
// перед каждым кейсом гасятся, чтобы окружение прогона не влияло на результат
var pgEnvKeys = []string{"PG_URL", "PG_PASSWORD", "PG_DBNAME", "PG_TIMEZONE"}

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

// TestPostgresURL строка подключения собирается из conf.yaml и переопределяется env
func TestPostgresURL(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "без env берутся conf.yaml и константы-дефолты",
			env:  nil,
			want: "postgresql://aggregator:aggregator@localhost:5432/aggregator" +
				"?sslmode=disable&timezone=Asia/Tashkent",
		},
		{
			name: "PG_URL переопределяет адрес",
			env:  map[string]string{"PG_URL": "db:5432"},
			want: "postgresql://aggregator:aggregator@db:5432/aggregator" +
				"?sslmode=disable&timezone=Asia/Tashkent",
		},
		{
			name: "PG_PASSWORD переопределяет пароль из conf.yaml",
			env:  map[string]string{"PG_PASSWORD": "s3cret"},
			want: "postgresql://aggregator:s3cret@localhost:5432/aggregator" +
				"?sslmode=disable&timezone=Asia/Tashkent",
		},
		{
			name: "PG_DBNAME переопределяет имя базы",
			env:  map[string]string{"PG_DBNAME": "aggregator_test"},
			want: "postgresql://aggregator:aggregator@localhost:5432/aggregator_test" +
				"?sslmode=disable&timezone=Asia/Tashkent",
		},
		{
			name: "PG_TIMEZONE переопределяет таймзону",
			env:  map[string]string{"PG_TIMEZONE": "UTC"},
			want: "postgresql://aggregator:aggregator@localhost:5432/aggregator" +
				"?sslmode=disable&timezone=UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range pgEnvKeys {
				t.Setenv(key, "")
			}

			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			c, err := config.NewConfig("conf.yaml")
			if err != nil {
				t.Fatalf("не удалось загрузить конфиг, ошибка %v", err)
			}

			if got := c.PostgresURL(); got != tt.want {
				t.Fatalf("PostgresURL() = %q, ожидалось %q", got, tt.want)
			}
		})
	}
}
