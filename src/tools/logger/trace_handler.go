package logger

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// keyTraceID ключ атрибута trace_id в структурных логах
const keyTraceID = "trace_id"

// keySpanID ключ атрибута span_id в структурных логах
const keySpanID = "span_id"

// traceHandler добавляет trace_id/span_id из активного спана ctx в каждую
// запись — связывает логи с трейсами в Jaeger. Если в ctx нет валидного
// спана (трейсинг выключен или вызов вне обёрнутого кода), атрибуты не
// добавляются
type traceHandler struct {
	slog.Handler
}

func (h traceHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String(keyTraceID, sc.TraceID().String()),
			slog.String(keySpanID, sc.SpanID().String()),
		)
	}

	return h.Handler.Handle(ctx, r)
}

// WithAttrs и WithGroup переопределены явно: без них promoted-методы
// эмбеднутого slog.Handler вернули бы его исходный тип и потеряли
// переопределенный Handle на всех дочерних логгерах (logger.NewUsecaseLogger)

func (h traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return traceHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h traceHandler) WithGroup(name string) slog.Handler {
	return traceHandler{Handler: h.Handler.WithGroup(name)}
}
