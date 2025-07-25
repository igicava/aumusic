package service

import (
	"aumusic/internal/config"
	"aumusic/internal/models"
	"aumusic/internal/repo"
	"aumusic/pkg/hash"
	"aumusic/pkg/logger"
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	MEDIA         = "/media/"
	MUSIC         = "/media/music/"
	MaxUploadSize = 500 << 20
)

var Pool *pgxpool.Pool

func ValidToken(ctx context.Context, token string) (username string, userid string, err error) {
	if token == "" {
		return "", "", http.ErrServerClosed
	}
	t, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, http.ErrServerClosed
		}
		return []byte(ctx.Value("cfg").(*config.Config).JWTSecret), nil
	})
	if err != nil {
		return "", "", err
	}
	if claims, ok := t.Claims.(jwt.MapClaims); ok && t.Valid {
		return claims["username"].(string), claims["userid"].(string), nil
	}
	return "", "", http.ErrServerClosed
}

func GetTrack(ctx context.Context, token, id string) (io.ReadSeeker, int64, time.Time, error) {
	track, err := repo.GetTrack(ctx, Pool, id)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to get track", zap.Error(err))
		return nil, 0, time.Time{}, http.ErrServerClosed
	}

	_, ownerId, err := ValidToken(ctx, token)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to validate token", zap.Error(err))
		return nil, 0, time.Time{}, err
	}
	if ownerId != track.UserId {
		return nil, 0, time.Time{}, http.ErrServerClosed
	}

	file, err := os.Open(track.Path)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to open file", zap.Error(err))
		return nil, 0, time.Time{}, err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to stat file", zap.Error(err))
		return nil, 0, time.Time{}, err
	}
	fileSize := fileInfo.Size()

	return file, fileSize, fileInfo.ModTime(), nil
}

func RegisterUser(ctx context.Context, r *http.Request) error {
	name := r.FormValue("username")
	email := r.FormValue("email")
	pass1 := r.FormValue("pass1")
	pass2 := r.FormValue("pass2")
	if pass1 != pass2 {
		return http.ErrServerClosed
	}
	passHash, err := hash.HashPasswordSecure(pass1)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to hash password", zap.Error(err))
		return err
	}
	err = repo.NewUser(ctx, Pool, models.User{
		Name:  name,
		Email: email,
		Pass:  passHash})
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to create user", zap.Error(err))
		return err
	}
	return nil
}

func LoginUser(ctx context.Context, r *http.Request) (string, error) {
	name := r.FormValue("name")
	pass := r.FormValue("pass")
	user, err := repo.GetUser(ctx, Pool, name)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to get user", zap.Error(err))
		return "", err
	}

	if isValid := hash.AuthenticateUser(user.Pass, pass); isValid != nil {
		return "", isValid
	}

	fmt.Println(r.FormValue("username"), r.FormValue("id"))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Name,
		"userid":   user.Id,
	})

	tokenString, err := token.SignedString([]byte(r.Context().Value("cfg").(*config.Config).JWTSecret))
	if err != nil {
		logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Failed to sign token", zap.Error(err))
		return "", err
	}

	return tokenString, nil
}

func GetTracksByUser(ctx context.Context, userId string) ([]models.Track, error) {
	tracks, err := repo.GetTracksByUser(ctx, Pool, userId)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to get tracks", zap.Error(err))
		return nil, err
	}
	return tracks, nil
}

func DeleteTrack(ctx context.Context, token, id string) error {
	track, err := repo.GetTrack(ctx, Pool, id)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to get track", zap.Error(err))
		return http.ErrServerClosed
	}

	owner, _, err := ValidToken(ctx, token)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to validate token", zap.Error(err))
		return err
	}
	if owner != track.UserId {
		return http.ErrServerClosed
	}

	err = os.Remove(track.Path)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to open file", zap.Error(err))
		return err
	}

	err = repo.DeleteTrack(ctx, Pool, id)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to delete track", zap.Error(err))
		return err
	}

	return nil
}
