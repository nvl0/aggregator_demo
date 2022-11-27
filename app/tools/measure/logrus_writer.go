package measure

import "github.com/sirupsen/logrus"

type logrusWriter struct {
	log *logrus.Logger
}

// NewLogrusWriter писать в логи logrus
func NewLogrusWriter(log *logrus.Logger) Writer {
	return &logrusWriter{log: log}
}

func (l *logrusWriter) Write(msg string) {
	l.log.Debugln(msg)
}
