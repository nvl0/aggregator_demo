// Package logger настраивает структурные логи приложения на log/slog.
package logger

import (
	"io"
	"log/slog"
	"os"
)

// newHandler собирает JSON-хендлер slog: source в каждой записи,
// уровень Debug при DEBUG=true, иначе Info
func newHandler(w io.Writer) slog.Handler {
	level := slog.LevelInfo
	if os.Getenv("DEBUG") == "true" {
		level = slog.LevelDebug
	}

	return slog.NewJSONHandler(w, &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	})
}

// New основной логгер приложения: JSON в stdout, с source, уровень по DEBUG
func New() *slog.Logger {
	return slog.New(newHandler(os.Stdout))
}

// NewDiscard логгер-заглушка для тестов
func NewDiscard() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
