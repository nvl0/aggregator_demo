package config

import (
	"fmt"
	"os"

	"github.com/jinzhu/configor"
)

// Config конфиг
type Config struct {
	Postgres struct {
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		URL      string `yaml:"url"`
	} `yaml:"postgres"`
}

// NewConfig инициализация конфига проекта
func NewConfig(confPath string) (Config, error) {
	var c = Config{}
	err := configor.Load(&c, confPath)
	return c, err
}

// PostgresURL адрес постгреса
func (c *Config) PostgresURL() string {
	pgUrl := os.Getenv("PG_URL")
	if pgUrl != "" {
		c.Postgres.URL = pgUrl
	}
	return fmt.Sprintf("postgresql://%s:%s@%s/aggregator?sslmode=disable&timezone=Asia/Tashkent",
		c.Postgres.User, c.Postgres.Password, c.Postgres.URL)
}
