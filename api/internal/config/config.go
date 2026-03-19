package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type DBConfig struct {
	URL string `env:"DATABASE_URL,required"`
}

type RedisConfig struct {
	URL string `env:"REDIS_URL,required"`
}

type GCSConfig struct {
	Bucket   string `env:"GCS_BUCKET,required"`
	Endpoint string `env:"GCS_ENDPOINT"` // override for fake-gcs-server in dev
}

type Config struct {
	App    AppConfig    `envPrefix:""`
	Server ServerConfig `envPrefix:""`
	DB     DBConfig     `envPrefix:""`
	Redis  RedisConfig  `envPrefix:""`
	GCS    GCSConfig    `envPrefix:""`
}

func Load() (*Config, error) {
	if os.Getenv("APP_ENV") == "dev" {
		if envFile := os.Getenv("ENV_FILE"); envFile != "" {
			if err := godotenv.Load(envFile); err != nil {
				return nil, err
			}
		}
	}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Env vars often store PEM keys with literal \n instead of real newlines.
	cfg.App.GithubAppPrivateKey = strings.ReplaceAll(cfg.App.GithubAppPrivateKey, `\n`, "\n")

	return &cfg, nil
}
