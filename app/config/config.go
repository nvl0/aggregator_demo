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
		Port     string `yaml:"port"`
	} `yaml:"postgres"`
}

// NewConfig init and return project config
func NewConfig(confPath string) (Config, error) {
	var c = Config{}
	err := configor.Load(&c, confPath)
	return c, err
}

func (c *Config) PostgresURL() string {
	pgUrl := os.Getenv("POSTGRES_IP_PORT")
	if pgUrl == "" {
		pgUrl = c.Postgres.URL
	}
	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable&timezone=Asia/Tashkent", c.Postgres.User, c.Postgres.Password, pgUrl, c.Postgres.User)
}
