package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		Server
		Postgres
		Logger
	}

	Server struct {
		Port            string        `yaml:"port" env:"SRV_PORT"`
		ReadTimeout     time.Duration `yaml:"read_timeout" env:"SRV_READ_TIMEOUT"`
		WriteTimeout    time.Duration `yaml:"write_timeout" env:"SRV_WRITE_TIMEOUT"`
		ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"SRV_SHUTDOWN_TIMEOUT"`
	}

	Postgres struct {
		User         string        `yaml:"user" env:"PG_USER"`
		Password     string        `env:"PG_PASSWORD"`
		Host         string        `yaml:"host" env:"PG_HOST"`
		Port         string        `yaml:"port" env:"PG_PORT"`
		Database     string        `yaml:"database" env:"PG_DATABASE"`
		SSLMode      string        `yaml:"sslmode" env:"PG_SSLMODE"`
		ConnAttempts int           `yaml:"conn_attempts" env:"PG_CONN_ATTEMPTS"`
		ConnTimeout  time.Duration `yaml:"conn_timeout" env:"PG_CONN_TIMEOUT"`
		MaxPoolSize  int           `yaml:"max_pool_size" env:"PG_MAX_POOL_SIZE"`
	}

	Logger struct {
		Level string `yaml:"level" env:"LOGGER_LEVEL"`
	}
)

func New(path string) (*Config, error) {
	config := new(Config)
	if err := cleanenv.ReadConfig(path, config); err != nil {
		return nil, err
	}

	if err := cleanenv.ReadEnv(config); err != nil {
		return nil, err
	}

	return config, nil
}
