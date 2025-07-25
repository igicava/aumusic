package postgres

import (
	"context"

	"errors"
	"fmt"

	"github.com/golang-migrate/migrate"
	"github.com/jackc/pgx/v5/pgxpool"

	"aumusic/pkg/logger"
	"go.uber.org/zap"

	_ "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
)

type Config struct {
	Host     string `yaml:"POSTGRES_HOST" env:"POSTGRES_HOST" env-default:"localhost"`
	Port     int    `yaml:"POSTGRES_PORT" env:"POSTGRES_PORT" env-default:"5432"`
	Username string `yaml:"POSTGRES_USER" env:"POSTGRES_USER" env-default:"root"`
	Password string `yaml:"POSTGRES_PASSWORD" env:"POSTGRES_PASSWORD" env-default:"1234"`
	Database string `yaml:"POSTGRES_DB" env:"POSTGRES_DB" env-default:"postgres"`
}

var Pool *pgxpool.Pool

func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)
	fmt.Println(connString)
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Fatal(ctx, "Failed to connect to postgres", zap.Error(err))
		return nil, err
	}
	m, err := migrate.New(
		"file://db/migrations",
		connString,
	)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Fatal(ctx, "Failed to connect to postgres", zap.Error(err))
		return nil, err
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logger.GetLoggerFromCtx(ctx).Fatal(ctx, "Failed to run migrations", zap.Error(err))
		return nil, err
	}

	err = pool.Ping(ctx)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Fatal(ctx, "Failed to ping postgres", zap.Error(err))
	}

	logger.GetLoggerFromCtx(ctx).Info(ctx, "Successfully connected to postgres")

	return pool, nil
}
