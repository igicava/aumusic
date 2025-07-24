package http

import (
	"aumusic/internal/config"
	"aumusic/pkg/logger"
	"context"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"net/http"
	"time"

	"aumusic/internal/server/http/handler"
)

func Run(ctx context.Context, cfg *config.Config) error {
	logger.GetLoggerFromCtx(ctx).Info(ctx, "Starting http server", zap.String("port", cfg.Port))
	r := http.NewServeMux()

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestId := uuid.NewString()
			ctx = context.WithValue(ctx, logger.RequestId, requestId)
			ctx = context.WithValue(ctx, "cfg", cfg)
			logger.GetLoggerFromCtx(ctx).Info(
				ctx,
				"request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Time("time", time.Now()))

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	r.HandleFunc("/", handler.Index)
	r.HandleFunc("/tracks/{id}", handler.RunTrack)
	r.HandleFunc("/tracks", handler.GetTracksByUser)
	r.HandleFunc("/register", handler.RegisterUser)
	r.HandleFunc("/login", handler.LoginUser)
	r.HandleFunc("/logout", handler.LogoutUser)

	mux := middleware(r)

	return http.ListenAndServe(":"+cfg.Port, mux)
}
