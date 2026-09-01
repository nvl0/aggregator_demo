// Package health http-пробы живости и готовности сервиса.
package health

import (
	"context"
	"net/http"
	"time"
)

const (
	// readyTimeout предельное время ожидания ответа бд в пробе готовности
	readyTimeout = 2 * time.Second

	// contentType тип тела ответа проб
	contentType = "text/plain; charset=utf-8"

	// bodyOK тело успешного ответа
	bodyOK = "ok"
	// bodyDBUnavailable тело ответа при недоступной бд
	bodyDBUnavailable = "db unavailable"
)

// Pinger источник проверки доступности бд.
// *sqlx.DB удовлетворяет интерфейсу, поэтому мок для тестов не нужен
type Pinger interface {
	PingContext(ctx context.Context) error
}

// Live проба живости: процесс поднят и обслуживает http.
// Никогда не ходит в бд, поэтому годится для liveness-пробы оркестратора
func Live(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(bodyOK))
}

// Ready проба готовности: 200 при доступной бд, 503 при недоступной.
// Пробу нельзя вешать как liveness — она конкурирует за соединения
// ограниченного пула с рабочим циклом агрегации
func Ready(db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
		defer cancel()

		w.Header().Set("Content-Type", contentType)

		if err := db.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(bodyDBUnavailable))

			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(bodyOK))
	}
}
