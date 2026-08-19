package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/jinzhu/configor"
)

// defaultWorkerPoolSize размер пула воркеров агрегации по умолчанию
const defaultWorkerPoolSize = 10

// Config конфиг
type Config struct {
	Postgres struct {
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		URL      string `yaml:"url"`
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

// PostgresURL адрес постгреса
func (c *Config) PostgresURL() string {
	pgURL := os.Getenv("PG_URL")
	if pgURL != "" {
		c.Postgres.URL = pgURL
	}
	return fmt.Sprintf("postgresql://%s:%s@%s/aggregator?sslmode=disable&timezone=Asia/Tashkent",
		c.Postgres.User, c.Postgres.Password, c.Postgres.URL)
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
