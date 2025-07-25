package handler

import (
	"aumusic/internal/models"
	"aumusic/internal/repo"
	"aumusic/internal/service"
	"aumusic/pkg/logger"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func enableCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func Index(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	cookie, err := r.Cookie("token")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if cookie.Value == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	http.ServeFile(w, r, "frontend/player.html")
}

func RunTrack(w http.ResponseWriter, r *http.Request) {
	trackName := r.PathValue("id")
	token, err := r.Cookie("token")
	if err != nil {
		logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Failed to get token", zap.Error(err))
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	file, fileSize, modTime, err := service.GetTrack(r.Context(), token.Value, trackName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		logger.GetLoggerFromCtx(r.Context()).Info(
			r.Context(),
			"Failed to get track",
			zap.String("trackName", trackName),
			zap.Error(err))
		return
	}

	enableCORS(&w)
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileSize))
	w.Header().Set("Accept-Ranges", "bytes")

	http.ServeContent(w, r, trackName, modTime, file)
}

func RegisterUser(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "GET" {
		http.ServeFile(w, r, "frontend/register.html")
		return
	}
	if r.Method == "POST" {
		err := service.RegisterUser(r.Context(), r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Failed to register user", zap.Error(err))
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func LoginUser(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method == "GET" {
		http.ServeFile(w, r, "frontend/login.html")
		return
	}
	if r.Method == "POST" {
		tokenString, err := service.LoginUser(r.Context(), r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Failed to login user", zap.Error(err))
			return
		}

		cookie := &http.Cookie{
			Name:     "token",
			Value:    tokenString,
			Path:     "/",
			HttpOnly: true,
		}
		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func LogoutUser(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	cookie := &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func GetTracksByUser(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	token, err := r.Cookie("token")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Failed to get token", zap.Error(err))
		return
	}
	_, userid, err := service.ValidToken(r.Context(), token.Value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Failed to validate token", zap.Error(err))
		return
	}
	tracks, err := service.GetTracksByUser(r.Context(), userid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Failed to get tracks", zap.Error(err))
		return
	}
	js, err := json.Marshal(tracks)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "json marshal error", zap.Error(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func LoadTracks(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		token, err := r.Cookie("token")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Failed to get token", zap.Error(err))
			return
		}
		name, userid, err := service.ValidToken(r.Context(), token.Value)
		fmt.Println(name, userid)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Failed to validate token", zap.Error(err))
			return
		}
		// Проверка размера запроса
		r.Body = http.MaxBytesReader(w, r.Body, service.MaxUploadSize)
		if err := r.ParseMultipartForm(service.MaxUploadSize); err != nil {
			http.Error(w, "The uploaded file is too big. Please choose a file that's less than 500MB", http.StatusBadRequest)
			return
		}

		// Получаем данные формы
		artist := r.FormValue("artist")
		album := r.FormValue("album")

		if artist == "" || album == "" {
			logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Artist or album is empty")
			http.Error(w, "Artist and album are required", http.StatusBadRequest)
			return
		}

		// Создаем директорию для артиста и альбома
		artistPath := filepath.Join(service.MUSIC, name, sanitizeName(artist))
		albumPath := filepath.Join(artistPath, sanitizeName(album))
		if err := os.MkdirAll(albumPath, os.ModePerm); err != nil {
			logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Failed to create album directory", zap.Error(err))
			http.Error(w, "Failed to create album directory", http.StatusInternalServerError)
			return
		}

		// Обрабатываем загруженные файлы
		files := r.MultipartForm.File["files"]
		uploadResults := make([]string, 0, len(files))

		for _, fileHeader := range files {
			file, err := fileHeader.Open()
			if err != nil {
				logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Error retrieving the file", zap.Error(err))
				http.Error(w, "Error retrieving the file", http.StatusBadRequest)
				return
			}
			defer file.Close()

			// Проверяем тип файла (можно добавить больше проверок)
			buff := make([]byte, 512)
			if _, err = file.Read(buff); err != nil {
				logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Error reading file", zap.Error(err))
				http.Error(w, "Error reading file", http.StatusInternalServerError)
				return
			}

			if _, err = file.Seek(0, io.SeekStart); err != nil {
				logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Error seeking file", zap.Error(err))
				http.Error(w, "Error seeking file", http.StatusInternalServerError)
				return
			}

			// Создаем файл на сервере
			dstPath := filepath.Join(albumPath, sanitizeName(fileHeader.Filename))
			dst, err := os.Create(dstPath)
			if err != nil {
				logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Error creating file on server", zap.Error(err))
				http.Error(w, "Error creating file on server", http.StatusInternalServerError)
				return
			}
			defer dst.Close()

			err = repo.AddTrack(r.Context(), service.Pool, models.TrackDB{
				UserId:  userid,
				Artist:  artist,
				Album:   album,
				Name:    fileHeader.Filename,
				Path:    dstPath,
				Size:    fileHeader.Size,
				ModTime: time.Now(),
			})
			if err != nil {
				logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Error creating file on server", zap.Error(err))
				http.Error(w, "Error creating file on server", http.StatusInternalServerError)
				return
			}

			// Копируем содержимое файла
			if _, err = io.Copy(dst, file); err != nil {
				logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Error saving file", zap.Error(err))
				http.Error(w, "Error saving file", http.StatusInternalServerError)
				return
			}

			uploadResults = append(uploadResults, fmt.Sprintf("Successfully uploaded %s (%d bytes)", fileHeader.Filename, fileHeader.Size))
		}

		// Возвращаем JSON-ответ
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{
			"status": "success",
			"message": "Upload complete",
			"details": {
				"artist": "%s",
				"album": "%s",
				"files_uploaded": %d,
				"results": %v
			}
		}`, artist, album, len(files), uploadResults)
		return
	}
	if r.Method == "GET" {
		token, err := r.Cookie("token")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Failed to get token", zap.Error(err))
			return
		}
		_, _, err = service.ValidToken(r.Context(), token.Value)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Failed to validate token", zap.Error(err))
			return
		}
		http.ServeFile(w, r, "frontend/upload.html")
		return
	}

}

func sanitizeName(name string) string {
	// Удаляем небезопасные символы из имени файла/папки
	return filepath.Base(name)
}

func DeleteTrack(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)
	if r.Method != "DELETE" {
		token, err := r.Cookie("token")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Failed to get token", zap.Error(err))
			return
		}
		trackId := r.PathValue("id")
		err = service.DeleteTrack(r.Context(), token.Value, trackId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Failed to delete track", zap.Error(err))
			return
		}
	}
}
