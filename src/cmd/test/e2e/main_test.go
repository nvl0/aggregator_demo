// Package e2e_test проверяет, что собранный бинарь агрегатора целиком —
// cron, служебный http, graceful shutdown — работает как единое целое
// против реального Postgres. Юнит- и интеграционные тесты покрывают части
// по отдельности, но не их склейку в cmd/main.go
package e2e_test

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	// healthPollTimeout предельное время ожидания готовности бинаря
	healthPollTimeout = 10 * time.Second
	// healthPollInterval пауза между попытками опроса
	healthPollInterval = 100 * time.Millisecond
	// shutdownWaitTimeout предельное время ожидания завершения после SIGTERM.
	// больше суммы таймаутов httpsrv (5s) и трейсера (5s) с запасом
	shutdownWaitTimeout = 15 * time.Second
)

// freeAddr занимает свободный порт, освобождает его и возвращает адрес
func freeAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr := l.Addr().String()
	require.NoError(t, l.Close())

	return addr
}

// buildBinary собирает бинарь агрегатора во временный файл
func buildBinary(t *testing.T) string {
	t.Helper()

	binPath := filepath.Join(t.TempDir(), "aggregator")

	cmd := exec.Command("go", "build", "-o", binPath, "aggregator/src/cmd")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "сборка бинаря упала: %s", out)

	return binPath
}

// prepareEnv готовит переменные окружения запуска: временные flow/subnet
// директории, реальный conf.yaml, адрес служебного http на свободном порту
func prepareEnv(t *testing.T, metricsAddr string) []string {
	t.Helper()

	flowDir := t.TempDir()

	subnetDir := t.TempDir()
	// пустой файл: без разрешенных исключений, но и без ошибки чтения
	require.NoError(t, os.WriteFile(filepath.Join(subnetDir, "internal"), nil, 0o644))

	confPath, err := filepath.Abs("../../../config/conf.yaml")
	require.NoError(t, err)

	pgURL := os.Getenv("PG_URL")
	if pgURL == "" {
		pgURL = "localhost:5432"
	}

	return []string{
		"CONF_PATH=" + confPath,
		"FLOW_DIR=" + flowDir,
		"SUBNET_DISABLED_DIR=" + subnetDir,
		"PG_URL=" + pgURL,
		"METRICS_ADDR=" + metricsAddr,
		"DEBUG=true",
		"VERSION=e2e-test",
	}
}

// waitForOK опрашивает url, пока он не ответит 200. Параллельно следит за
// done: если процесс завершился раньше готовности, тест падает сразу
// с содержимым его вывода, а не по общему таймауту
func waitForOK(t *testing.T, url string, done <-chan error, output *bytes.Buffer) {
	t.Helper()

	deadline := time.Now().Add(healthPollTimeout)

	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			t.Fatalf("процесс завершился раньше готовности %s, ошибка %v, вывод:\n%s",
				url, err, output.String())
		default:
		}

		resp, err := http.Get(url) // тестовый опрос локального адреса
		if err == nil {
			code := resp.StatusCode
			_ = resp.Body.Close()

			if code == http.StatusOK {
				return
			}
		}

		time.Sleep(healthPollInterval)
	}

	t.Fatalf("%s не ответил 200 за %v, вывод:\n%s", url, healthPollTimeout, output.String())
}

func TestMainBinaryServesAndShutsDownGracefully(t *testing.T) {
	binPath := buildBinary(t)
	metricsAddr := freeAddr(t)

	cmd := exec.Command(binPath) // путь к бинарю собран этим же тестом
	cmd.Env = prepareEnv(t, metricsAddr)

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	require.NoError(t, cmd.Start())

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	defer func() {
		// на случай падения теста раньше SIGTERM — не оставляем процесс висеть.
		// после штатного завершения Kill на уже мертвом процессе безопасен
		_ = cmd.Process.Kill()
	}()

	base := "http://" + metricsAddr

	waitForOK(t, base+"/healthz", done, &output)
	waitForOK(t, base+"/readyz", done, &output)

	resp, err := http.Get(base + "/metrics") // тестовый запрос к локальному адресу
	require.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, resp.Body.Close())
	require.NoError(t, err)
	require.Contains(t, string(body), "aggregator_build_info",
		"метрика сборки должна быть в выдаче /metrics")

	require.NoError(t, cmd.Process.Signal(syscall.SIGTERM))

	select {
	case waitErr := <-done:
		require.NoError(t, waitErr, "бинарь должен завершиться без ошибки, вывод:\n%s", output.String())
	case <-time.After(shutdownWaitTimeout):
		t.Fatalf("бинарь не завершился за %v после SIGTERM, вывод:\n%s",
			shutdownWaitTimeout, output.String())
	}
}
