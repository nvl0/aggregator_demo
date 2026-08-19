package measure

import (
	"log/slog"
	"os"
)

type fmtWriter struct {
	log *slog.Logger
}

// NewFmtWriter писать в логи fmt
func NewFmtWriter() Writer {
	return &fmtWriter{log: slog.New(slog.NewTextHandler(os.Stderr, nil))}
}

func (l *fmtWriter) Write(msg string) {
	l.log.Info(msg)
}
