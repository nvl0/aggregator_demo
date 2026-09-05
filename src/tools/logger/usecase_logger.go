package logger

import "log/slog"

// NewUsecaseLogger дочерний логгер юзкейса с атрибутом component.
// New() не ставит базовых атрибутов, поэтому component попадает в запись
// ровно один раз
func NewUsecaseLogger(parent *slog.Logger, component string) *slog.Logger {
	return parent.With("component", component)
}
