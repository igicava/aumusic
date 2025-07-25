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
	"path/filepath"
	"time"
)

const (
	MEDIA         = "/media/"
	MUSIC         = "/media/music/"
	MaxUploadSize = 100 << 20
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

	owner, _, err := ValidToken(ctx, token)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to validate token", zap.Error(err))
		return nil, 0, time.Time{}, err
	}
	if owner != track.UserId {
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

func LoginUser(ctx context.Context, r *http.Request) error {
	name := r.FormValue("name")
	pass := r.FormValue("pass")
	user, err := repo.GetUser(ctx, Pool, name)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to get user", zap.Error(err))
		return err
	}
	if isValid := hash.AuthenticateUser(user.Pass, pass); isValid != nil {
		return isValid
	}

	return nil
}

func GetTracksByUser(ctx context.Context, userId string) ([]models.Track, error) {
	tracks, err := repo.GetTracksByUser(ctx, Pool, userId)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to get tracks", zap.Error(err))
		return nil, err
	}
	return tracks, nil
}

func LoadTracks(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	token, err := r.Cookie("token")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusUnauthorized)
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to get token", zap.Error(err))
		return err
	}
	_, userid, err := ValidToken(ctx, token.Value)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusUnauthorized)
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to validate token", zap.Error(err))
		return err
	}

	artist, album := r.FormValue("artist"), r.FormValue("album")
	uploadPath := fmt.Sprintf("%s%s%s%s", MUSIC, userid, artist, album)

	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadSize)
	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to parse multipart form", zap.Error(err))
		http.Error(w, "File too large", http.StatusBadRequest)
		return err
	}

	files := r.MultipartForm.File["files"]
	for _, fileHeader := range files {
		// Открываем файл
		file, err := fileHeader.Open()
		if err != nil {
			logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to open file", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		defer file.Close()

		// Создаем целевой файл
		dstPath := filepath.Join(uploadPath, fileHeader.Filename)
		dst, err := os.Create(dstPath)
		if err != nil {
			logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to create file", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
		defer dst.Close()

		// Копируем содержимое файла
		if _, err := io.Copy(dst, file); err != nil {
			logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to copy file", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		fileInfo, err := dst.Stat()
		if err != nil {
			logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to stat file", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}

		track := models.TrackDB{
			UserId:  userid,
			Artist:  artist,
			Album:   album,
			Name:    fileHeader.Filename,
			Path:    dstPath,
			Size:    fileHeader.Size,
			ModTime: fileInfo.ModTime(),
		}

		err = repo.AddTrack(ctx, Pool, track)
		if err != nil {
			logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to add track", zap.Error(err))
			return err
		}
	}

	return nil
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
