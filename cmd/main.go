package main

import (
	"aumusic/internal/config"
	httpserver "aumusic/internal/server/http"
	"aumusic/internal/service"
	"aumusic/pkg/logger"
	"aumusic/pkg/minio"
	"aumusic/pkg/postgres"

	"context"
)

func main() {
	ctx, err := logger.New(context.Background())
	if err != nil {
		panic(err)
	}

	cfg, err := config.New()
	if err != nil {
		panic(err)
	}

	service.Pool, err = postgres.NewPool(ctx, cfg.Postgres)
	if err != nil {
		panic(err)
	}
	
	service.MinIO, err = minio.New(ctx, cfg.Minio.Endpoint, cfg.Minio.AccessKey, cfg.Minio.SecretKey, cfg.Minio.UseSSL)
	if err != nil {
		panic(err)
	}

	if err := httpserver.Run(ctx, cfg); err != nil {
		panic(err)
	}
}
