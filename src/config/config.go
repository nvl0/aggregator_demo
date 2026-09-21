package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/jinzhu/configor"
)

// defaultWorkerPoolSize размер пула воркеров агрегации по умолчанию
const defaultWorkerPoolSize = 10

// defaultMetricsAddr адрес служебного http сервера по умолчанию.
// :9090 занят самим prometheus, :2112 — конвенция client_golang
const defaultMetricsAddr = ":2112"

// defaultPostgresDBName имя базы по умолчанию
const defaultPostgresDBName = "aggregator"

// defaultPostgresTimezone таймзона соединения с бд по умолчанию
const defaultPostgresTimezone = "Asia/Tashkent"

// Config конфиг
type Config struct {
	Postgres struct {
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		URL      string `yaml:"url"`
		DBName   string `yaml:"db_name"`
		Timezone string `yaml:"timezone"`
	} `yaml:"postgres"`
	WorkerPool struct {
		Size int `yaml:"size"`
	} `yaml:"worker_pool"`
}

// NewConfig инициализация конфига проекта
func NewConfig(confPath string) (Config, error) {
	var c = Config{}
	err := configor.Load(&c, confPath)
	return c, err
}

// PostgresURL адрес постгреса.
// Значения из conf.yaml переопределяются переменными окружения
// PG_URL, PG_PASSWORD, PG_DBNAME, PG_TIMEZONE
func (c *Config) PostgresURL() string {
	if pgURL := os.Getenv("PG_URL"); pgURL != "" {
		c.Postgres.URL = pgURL
	}

	if pgPassword := os.Getenv("PG_PASSWORD"); pgPassword != "" {
		c.Postgres.Password = pgPassword
	}

	if pgDBName := os.Getenv("PG_DBNAME"); pgDBName != "" {
		c.Postgres.DBName = pgDBName
	}

	if pgTimezone := os.Getenv("PG_TIMEZONE"); pgTimezone != "" {
		c.Postgres.Timezone = pgTimezone
	}

	if c.Postgres.DBName == "" {
		c.Postgres.DBName = defaultPostgresDBName
	}

	if c.Postgres.Timezone == "" {
		c.Postgres.Timezone = defaultPostgresTimezone
	}

	return fmt.Sprintf("postgresql://%s:%s@%s/%s?sslmode=disable&timezone=%s",
		c.Postgres.User, c.Postgres.Password, c.Postgres.URL,
		c.Postgres.DBName, c.Postgres.Timezone)
}

// WorkerPoolSize размер пула воркеров агрегации.
// Значение из conf.yaml переопределяется переменной окружения WORKER_POOL_SIZE.
func (c *Config) WorkerPoolSize() int {
	if envSize := os.Getenv("WORKER_POOL_SIZE"); envSize != "" {
		if size, err := strconv.Atoi(envSize); err == nil && size > 0 {
			c.WorkerPool.Size = size
		}
	}

	if c.WorkerPool.Size < 1 {
		c.WorkerPool.Size = defaultWorkerPoolSize
	}

	return c.WorkerPool.Size
}

// MetricsAddr адрес служебного http сервера метрик и health-проб.
// Задается только переменной окружения METRICS_ADDR, в conf.yaml ключа нет
func (c *Config) MetricsAddr() string {
	if addr := os.Getenv("METRICS_ADDR"); addr != "" {
		return addr
	}

	return defaultMetricsAddr
}

// TracingEndpoint адрес otlp-коллектора трейсов.
// Задается только переменной окружения TRACING_ENDPOINT, в conf.yaml ключа нет.
// Пустое значение выключает трейсинг
func (c *Config) TracingEndpoint() string {
	return os.Getenv("TRACING_ENDPOINT")
}
