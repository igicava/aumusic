package main

import (
	httpserver "aumusic/internal/server/http"
	"aumusic/pkg/logger"
	"context"
	"log"
)

func main() {
	ctx, err := logger.New(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	if err := httpserver.Run(ctx); err != nil {
		panic(err)
	}
}
