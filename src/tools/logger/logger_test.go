package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"aggregator/src/tools/logger"

	"github.com/stretchr/testify/require"
)

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

func TestNewDiscardDropsAllLevels(t *testing.T) {
	r := require.New(t)

	l := logger.NewDiscard()
	r.False(l.Handler().Enabled(context.Background(), slog.LevelError))
	r.False(l.Handler().Enabled(context.Background(), slog.LevelDebug))
}
