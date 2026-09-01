package health_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"aggregator/src/external/health"
)

// stubPinger подконтрольная реализация health.Pinger
type stubPinger struct {
	err error
}

func (s stubPinger) PingContext(_ context.Context) error {
	return s.err
}

func TestLive(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	health.Live(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("код ответа = %d, ожидался %d", rec.Code, http.StatusOK)
	}

	if body := rec.Body.String(); body != "ok" {
		t.Errorf("тело ответа = %q, ожидалось %q", body, "ok")
	}
}

func TestReady(t *testing.T) {
	tests := []struct {
		name     string
		pingErr  error
		wantCode int
	}{
		{
			name:     "бд отвечает, готовность подтверждена",
			pingErr:  nil,
			wantCode: http.StatusOK,
		},
		{
			name:     "бд недоступна, готовность не подтверждена",
			pingErr:  errors.New("соединение отклонено"),
			wantCode: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := health.Ready(stubPinger{err: tt.pingErr})

			req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("код ответа = %d, ожидался %d", rec.Code, tt.wantCode)
			}
		})
	}
}
