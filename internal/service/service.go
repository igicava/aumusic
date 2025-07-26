package service

import (
	"aumusic/internal/config"
	"aumusic/internal/models"
	"aumusic/internal/repo"
	"aumusic/pkg/hash"
	"aumusic/pkg/logger"
	"context"
	"errors"
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
	MaxUploadSize = 500 << 20
)

var Pool *pgxpool.Pool

func ValidToken(ctx context.Context, token string) (string, string, error) {
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
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")
	passHash, err := hash.GenerateHash(password, hash.DefaultArgon2Params)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to hash password", zap.Error(err))
		return err
	}
	err = repo.NewUser(ctx, Pool, models.User{
		Username: username,
		Email:    email,
		Pass:     passHash,
	})
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to create user", zap.Error(err))
		return err
	}
	return nil
}

func LoginUser(ctx context.Context, r *http.Request) (string, error) {
	username := r.FormValue("username")
	pass := r.FormValue("password")
	user, err := repo.GetUser(ctx, Pool, username)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to get user", zap.Error(err))
		return "", err
	}

	if isValid, _ := hash.VerifyPassword(pass, user.Pass); !isValid {
		return "", errors.New("invalid password")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
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

	_, ownerId, err := ValidToken(ctx, token)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to validate token", zap.Error(err))
		return err
	}
	if ownerId != track.UserId {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "User is not owner of track")
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

func LoadTracks(ctx context.Context, r *http.Request, artist, album, name, userid string) (int, []string, int, error) {
	if artist == "" || album == "" {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Artist or album is empty")
		return http.StatusBadRequest, []string{}, 0, errors.New("artist and album are required")
	}

	// Создаем директорию для артиста и альбома
	artistPath := filepath.Join(MUSIC, name, sanitizeName(artist))
	albumPath := filepath.Join(artistPath, sanitizeName(album))
	if err := os.MkdirAll(albumPath, os.ModePerm); err != nil {
		logger.GetLoggerFromCtx(ctx).Info(ctx, "Failed to create album directory", zap.Error(err))
		return http.StatusInternalServerError, []string{}, 0, err
	}

	// Обрабатываем загруженные файлы
	files := r.MultipartForm.File["files"]
	uploadResults := make([]string, 0, len(files))

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			logger.GetLoggerFromCtx(ctx).Info(ctx, "Error retrieving the file", zap.Error(err))
			return http.StatusBadRequest, []string{}, 0, err
		}
		defer file.Close()

		// Проверяем тип файла (можно добавить больше проверок)
		buff := make([]byte, 512)
		if _, err = file.Read(buff); err != nil {
			logger.GetLoggerFromCtx(ctx).Info(ctx, "Error reading file", zap.Error(err))
			return http.StatusInternalServerError, []string{}, 0, err
		}

		if _, err = file.Seek(0, io.SeekStart); err != nil {
			logger.GetLoggerFromCtx(ctx).Info(ctx, "Error seeking file", zap.Error(err))
			return http.StatusInternalServerError, []string{}, 0, err
		}

		// Создаем файл на сервере
		dstPath := filepath.Join(albumPath, sanitizeName(fileHeader.Filename))
		dst, err := os.Create(dstPath)
		if err != nil {
			logger.GetLoggerFromCtx(ctx).Info(ctx, "Error creating file on server", zap.Error(err))
			return http.StatusInternalServerError, []string{}, 0, err
		}
		defer dst.Close()

		err = repo.AddTrack(ctx, Pool, models.TrackDB{
			UserId:  userid,
			Artist:  artist,
			Album:   album,
			Name:    fileHeader.Filename,
			Path:    dstPath,
			Size:    fileHeader.Size,
			ModTime: time.Now(),
		})
		if err != nil {
			logger.GetLoggerFromCtx(ctx).Info(ctx, "Error creating file on server", zap.Error(err))
			return http.StatusInternalServerError, []string{}, 0, err
		}

		// Копируем содержимое файла
		if _, err = io.Copy(dst, file); err != nil {
			logger.GetLoggerFromCtx(ctx).Info(ctx, "Error saving file", zap.Error(err))
			return http.StatusInternalServerError, []string{}, 0, err
		}

		uploadResults = append(uploadResults, fmt.Sprintf("Successfully uploaded %s (%d bytes)", fileHeader.Filename, fileHeader.Size))
	}

	return http.StatusOK, uploadResults, len(files), nil
}

func sanitizeName(name string) string {
	// Удаляем небезопасные символы из имени файла/папки
	return filepath.Base(name)
}
