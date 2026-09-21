package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"aggregator/src/tools/logger"
	"aggregator/src/tools/tracing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

// TestMain глушит дефолтный error-handler otel: в тестах экспорт на
// заведомо пустой endpoint намеренно падает при shutdown, это не ошибка теста
func TestMain(m *testing.M) {
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(error) {}))

	os.Exit(m.Run())
}

func TestNewHandlerWritesSourceAndLevel(t *testing.T) {
	r := require.New(t)

	buf := &bytes.Buffer{}
	l := slog.New(logger.NewHandlerForTest(buf))
	l.Info("тестовая запись")

	var entry map[string]any
	r.NoError(json.Unmarshal(buf.Bytes(), &entry))

	_, hasSource := entry["source"]
	r.True(hasSource, "запись должна содержать ключ source")
	r.Equal("INFO", entry["level"])
}

func TestNewHandlerLevelByDebugEnv(t *testing.T) {
	r := require.New(t)

	t.Setenv("DEBUG", "true")
	h := logger.NewHandlerForTest(os.Stdout)
	r.True(h.Enabled(context.Background(), slog.LevelDebug))

	t.Setenv("DEBUG", "")
	h = logger.NewHandlerForTest(os.Stdout)
	r.False(h.Enabled(context.Background(), slog.LevelDebug))
}

func TestNewUsecaseLoggerAddsComponent(t *testing.T) {
	r := require.New(t)

	buf := &bytes.Buffer{}
	base := slog.New(logger.NewHandlerForTest(buf))
	child := logger.NewUsecaseLogger(base, "flow")
	child.Info("тестовая запись")

	var entry map[string]any
	r.NoError(json.Unmarshal(buf.Bytes(), &entry))
	r.Equal("flow", entry["component"])
}

func TestHandlerAddsTraceAttributesWhenSpanValid(t *testing.T) {
	r := require.New(t)

	buf := &bytes.Buffer{}
	l := slog.New(logger.NewHandlerForTest(buf))

	tr, shutdown := tracing.New(context.Background(), logger.NewDiscard(), "127.0.0.1:4318")
	defer func() { _ = shutdown(context.Background()) }()

	ctx, span := tr.Start(context.Background(), "op")
	defer span.End()

	l.InfoContext(ctx, "тестовая запись")

	var entry map[string]any
	r.NoError(json.Unmarshal(buf.Bytes(), &entry))
	r.Equal(span.SpanContext().TraceID().String(), entry["trace_id"])
	r.Equal(span.SpanContext().SpanID().String(), entry["span_id"])
}

func TestHandlerOmitsTraceAttributesWhenNoSpan(t *testing.T) {
	r := require.New(t)

	buf := &bytes.Buffer{}
	l := slog.New(logger.NewHandlerForTest(buf))

	l.InfoContext(context.Background(), "тестовая запись")

	var entry map[string]any
	r.NoError(json.Unmarshal(buf.Bytes(), &entry))
	_, hasTraceID := entry["trace_id"]
	r.False(hasTraceID, "без спана в контексте trace_id быть не должно")
}

func TestNewDiscardDropsAllLevels(t *testing.T) {
	r := require.New(t)

	l := logger.NewDiscard()
	r.False(l.Handler().Enabled(context.Background(), slog.LevelError))
	r.False(l.Handler().Enabled(context.Background(), slog.LevelDebug))
}
