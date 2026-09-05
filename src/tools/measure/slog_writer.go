package measure

import "log/slog"

type slogWriter struct {
	log *slog.Logger
}

// NewSlogWriter писать в логи slog
func NewSlogWriter(log *slog.Logger) Writer {
	return &slogWriter{log: log}
}

func (l *slogWriter) Write(text string) {
	l.log.Debug(text)
}
