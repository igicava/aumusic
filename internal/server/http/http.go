package http

import (
	"aumusic/pkg/logger"
	"context"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"net/http"
	"time"

	"aumusic/internal/server/http/handler"
)

func Run(ctx context.Context) error {
	r := http.NewServeMux()

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestId := uuid.NewString()
			ctx = context.WithValue(ctx, logger.RequestId, requestId)
			logger.GetLoggerFromCtx(ctx).Info(
				ctx,
				"request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Time("time", time.Now()))

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	r.HandleFunc("/tracks/{fileName}", handler.RunTrack)
	r.HandleFunc("/tracks", handler.ListTracks)

	mux := middleware(r)

	return http.ListenAndServe(":8080", mux)
}
