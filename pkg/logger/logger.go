package logger

import (
	"log/slog"
	"os"
	"strings"
)

var Logger *slog.Logger

func InitLogger(logLevel string) {
	if Logger != nil {
		return
	}
	level := slog.LevelInfo
	switch strings.ToLower(logLevel) {
	case "info":
		level = slog.LevelInfo
	case "debug":
		level = slog.LevelDebug
	case "error":
		level = slog.LevelError
	case "warn":
		level = slog.LevelWarn
	}
	Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}
