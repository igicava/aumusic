package handler

import (
	"aumusic/internal/service"
	"aumusic/pkg/logger"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"net/http"
)

func enableCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func RunTrack(w http.ResponseWriter, r *http.Request) {
	trackName := r.PathValue("fileName")

	file, fileSize, modTime, err := service.GetTrack(trackName)
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

func ListTracks(w http.ResponseWriter, r *http.Request) {
	tracksNames, err := service.ListTracks()
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
