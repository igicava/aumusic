package handler

import (
	"aumusic/internal/config"
	"aumusic/internal/service"
	"aumusic/pkg/logger"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"net/http"
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

// ListTracks is a legacy method
func ListTracks(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	tracksNames, err := service.ListTracks(r.Context(), path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Failed to list tracks", zap.Error(err))
		return
	}

	js, err := json.Marshal(tracksNames)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "json marshal error", zap.Error(err))
		return
	}

	enableCORS(&w)
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
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
		err := service.LoginUser(r.Context(), r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Failed to login user", zap.Error(err))
			return
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": r.FormValue("name"),
			"userid":   r.FormValue("id"),
		})
		tokenString, err := token.SignedString(r.Context().Value("cfg").(*config.Config).JWTSecret)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.GetLoggerFromCtx(r.Context()).Info(r.Context(), "Failed to sign token", zap.Error(err))
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
