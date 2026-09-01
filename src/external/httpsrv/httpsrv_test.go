package httpsrv_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"aggregator/src/external/httpsrv"
	"aggregator/src/tools/logger"
)

// waitTimeout предельное ожидание возврата Run в тестах
const waitTimeout = 5 * time.Second

var testLogger = logger.NewNoFileLogger("test")

// freeAddr занимает свободный порт, освобождает его и возвращает адрес
func freeAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("не удалось занять свободный порт, ошибка %v", err)
	}

	addr := l.Addr().String()

	if err = l.Close(); err != nil {
		t.Fatalf("не удалось освободить порт, ошибка %v", err)
	}

	return addr
}

// getWithRetry ходит по адресу, пока сервер не поднимется
func getWithRetry(t *testing.T, url string) *http.Response {
	t.Helper()

	deadline := time.Now().Add(waitTimeout)

	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			return resp
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("сервер не поднялся за %v", waitTimeout)

	return nil
}

func TestRunServesAndShutsDown(t *testing.T) {
	addr := freeAddr(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- httpsrv.Run(ctx, testLogger, addr, mux) }()

	resp := getWithRetry(t, "http://"+addr+"/ping")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("код ответа = %d, ожидался %d", resp.StatusCode, http.StatusOK)
	}
	_ = resp.Body.Close()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run вернул ошибку %v, ожидался nil", err)
		}
	case <-time.After(waitTimeout):
		t.Fatalf("Run не вернулся за %v после отмены контекста", waitTimeout)
	}
}

func TestRunReturnsListenError(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("не удалось занять порт, ошибка %v", err)
	}
	defer func() { _ = l.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	// порт уже занят, ListenAndServe обязан вернуть ошибку
	go func() { done <- httpsrv.Run(ctx, testLogger, l.Addr().String(), http.NewServeMux()) }()

	select {
	case runErr := <-done:
		if runErr == nil {
			t.Error("Run вернул nil, ожидалась ошибка занятого порта")
		}
	case <-time.After(waitTimeout):
		t.Fatalf("Run не вернулся за %v на занятом порту", waitTimeout)
	}
}
