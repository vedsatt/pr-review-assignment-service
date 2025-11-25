package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/vedsatt/pr-review-assignment-service/internal/repository"
)

type Config struct {
	repository.PostgresCfg

	HTTPPort string `env:"PORT" env-default:"8080"`
}

func NewConfig() (*Config, error) {
	var cfg Config

	path := os.Getenv("ENV_PATH")
	if path == "" {
		path = "./config/.env"
	}

	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return &cfg, err
}
