package minio

import (
	"aumusic/pkg/logger"
	"go.uber.org/zap"

	"context"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint  string `yaml:"endpoint" env:"MINIO_ENDPOINT" env-default:"localhost:9000"`
	AccessKey string `yaml:"access_key" env:"MINIO_ACCESS_KEY" env-default:"minio"`
	SecretKey string `yaml:"secret_key" env:"MINIO_SECRET_KEY" env-default:"minio123"`
	UseSSL    bool   `yaml:"use_ssl" env:"MINIO_USE_SSL" env-default:"false"`
}

func New(ctx context.Context, endpoint, accessKey, secretKey string, useSSL bool) (*minio.Client, error) {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Fatal(ctx, "Failed to connect to minio", zap.Error(err))
		return nil, err
	}

	log.Printf("%#v\n", minioClient) // minioClient is now set up

	return minioClient, nil
}
