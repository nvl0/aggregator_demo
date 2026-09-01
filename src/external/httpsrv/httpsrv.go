// Package httpsrv служебный http сервер с явными таймаутами и graceful shutdown.
package httpsrv

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	// readHeaderTimeout предельное время чтения заголовков запроса.
	// Явное значение закрывает Slowloris (gosec G112)
	readHeaderTimeout = 5 * time.Second
	// readTimeout предельное время чтения всего запроса
	readTimeout = 10 * time.Second
	// writeTimeout предельное время записи ответа
	writeTimeout = 10 * time.Second
	// idleTimeout предельное время простоя keep-alive соединения
	idleTimeout = 60 * time.Second
	// shutdownTimeout предельное время на доработку соединений при остановке
	shutdownTimeout = 5 * time.Second
)

// Run поднимает http сервер на addr и блокируется до отмены ctx.
// По отмене контекста выполняет graceful shutdown и возвращает его результат.
// Штатное закрытие сервера (http.ErrServerClosed) ошибкой не считается
func Run(ctx context.Context, log *logrus.Logger, addr string, h http.Handler) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	// буфер на 1: горутина не зависнет, если Run уже вышел по отмене контекста
	errChan := make(chan error, 1)

	go func() {
		log.Infoln("служебный http сервер запущен, адрес", addr)

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err

			return
		}

		close(errChan)
	}()

	select {
	case err := <-errChan:
		log.Errorln("служебный http сервер завершился, ошибка", err)

		return err
	case <-ctx.Done():
	}

	// контекст приложения уже отменен, поэтому на shutdown берем собственный
	sc, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	log.Infoln("остановка служебного http сервера")

	return srv.Shutdown(sc)
}
