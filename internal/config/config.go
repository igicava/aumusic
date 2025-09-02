package config

import (
	"aumusic/pkg/minio"
	"aumusic/pkg/postgres"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Postgres postgres.Config `yaml:"POSTGRES" env:"POSTGRES"`
	Minio    minio.Config    `yaml:"MINIO" env:"MINIO"`

	Port      string `yaml:"APP_PORT" env:"APP_PORT" env-default:"8081"`
	JWTSecret string `yaml:"JWT_SECRET" env:"JWT_SECRET" env-default:"secret"`
}

func New() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadConfig(".env", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
