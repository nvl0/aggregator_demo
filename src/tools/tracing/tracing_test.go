package tracing_test

import (
	"context"
	"os"
	"testing"

	"aggregator/src/tools/logger"
	"aggregator/src/tools/tracing"

	"go.opentelemetry.io/otel"
)

var testLogger = logger.NewDiscard()

// TestMain глушит дефолтный error-handler otel: в тестах экспорт на
// заведомо пустой endpoint намеренно падает при shutdown, это не ошибка теста
func TestMain(m *testing.M) {
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(error) {}))

	os.Exit(m.Run())
}

func TestNewWithEmptyEndpointReturnsNoopTracer(t *testing.T) {
	tr, shutdown := tracing.New(context.Background(), testLogger, "")
	defer func() { _ = shutdown(context.Background()) }()

	_, span := tr.Start(context.Background(), "op")
	defer span.End()

	if span.SpanContext().IsValid() {
		t.Error("пустой endpoint должен давать no-op трейсер, но спан оказался валидным")
	}
}

func TestNopReturnsInvalidSpans(t *testing.T) {
	tr := tracing.Nop()

	_, span := tr.Start(context.Background(), "op")
	defer span.End()

	if span.SpanContext().IsValid() {
		t.Error("Nop() должен давать no-op трейсер, но спан оказался валидным")
	}
}

func TestNewWithEndpointReturnsRecordingTracer(t *testing.T) {
	tr, shutdown := tracing.New(context.Background(), testLogger, "127.0.0.1:4318")
	defer func() { _ = shutdown(context.Background()) }()

	_, span := tr.Start(context.Background(), "op")
	defer span.End()

	if !span.SpanContext().IsValid() {
		t.Error("непустой endpoint должен давать реальный трейсер, но спан оказался невалидным")
	}

	if !span.IsRecording() {
		t.Error("спан от реального трейсера должен быть в состоянии записи")
	}
}
