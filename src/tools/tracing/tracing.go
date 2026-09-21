// Package tracing OpenTelemetry-трейсер агрегатора на собственном экземпляре,
// без глобального состояния (по аналогии с tools/metrics).
package tracing

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// attrServiceName ключ атрибута имени сервиса в ресурсе трейсера
const attrServiceName = "service.name"

// serviceName имя сервиса в атрибутах спанов и ресурсе трейсера
const serviceName = "aggregator"

// noShutdown shutdown-функция для no-op трейсера: закрывать нечего
func noShutdown(context.Context) error { return nil }

// Tracer трейсер агрегатора на собственном экземпляре trace.Tracer
type Tracer struct {
	tracer trace.Tracer
}

// Start открывает спан name, потомка спана из ctx (если он там есть).
// End спана — забота вызывающего кода, поэтому здесь не отслеживается
func (t *Tracer) Start(ctx context.Context, name string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name) //nolint:spancheck // End на стороне вызывающего
}

// nopTracer трейсер, создающий невалидные, ничего не пишущие спаны
func nopTracer() *Tracer {
	return &Tracer{tracer: noop.NewTracerProvider().Tracer(serviceName)}
}

// Nop трейсер, который ничего не пишет: дефолт там, где реальный трейсер
// не передан (тесты, loadgen), по аналогии с metrics.Nop()
func Nop() *Tracer {
	return nopTracer()
}

// New создает трейсер с экспортом в endpoint по OTLP/HTTP.
// endpoint пустой: трейсинг выключен, возвращается no-op трейсер.
// Ошибка создания экспортера не фатальна: логируется, трейсинг остается выключен
func New(ctx context.Context, log *slog.Logger, endpoint string) (*Tracer, func(context.Context) error) {
	if endpoint == "" {
		return nopTracer(), noShutdown
	}

	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		log.ErrorContext(ctx, "не удалось создать otlp-экспортёр трейсов, трейсинг отключен", "error", err)

		return nopTracer(), noShutdown
	}

	res := resource.NewSchemaless(
		attribute.String(attrServiceName, serviceName),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)

	return &Tracer{tracer: tp.Tracer(serviceName)}, tp.Shutdown
}
