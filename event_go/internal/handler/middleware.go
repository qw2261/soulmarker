package handler

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/qw2261/soulmarker/event_go/internal/config"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func init() {
	cfg := config.Load()
	levelStr := cfg.LogLevel

	var level slog.Level
	switch levelStr {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}
	if cfg.LogFormat == "text" {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, opts)))
	} else {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, opts)))
	}
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		latency := time.Since(start).Milliseconds()

		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.RemoteAddr
		}

		bodySize := r.ContentLength
		status := rw.statusCode

		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}

		slog.LogAttrs(r.Context(), level, "request",
			slog.Time("time", start),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", status),
			slog.Int64("latency_ms", latency),
			slog.String("ip", ip),
			slog.Int64("body_size", bodySize),
		)
	})
}
