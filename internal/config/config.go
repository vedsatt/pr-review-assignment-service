package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/vedsatt/pr-review-assignment-service/internal/repository"
	"go.uber.org/zap"
)

type Config struct {
	repository.PostgresCfg

	HTTPPort string `env:"PORT" env-default:"8080"`
}

func NewConfig() (*Config, error) {
	var cfg Config

	err := cleanenv.ReadConfig(".env", &cfg)
	if err != nil {
		zap.L().Warn(".env file not found, using default values")
		err = cleanenv.ReadEnv(&cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	return &cfg, err
}
