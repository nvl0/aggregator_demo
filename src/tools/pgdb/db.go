package pgdb

import (
	"log/slog"
	"os"

	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq" // регистрация драйвера postgres для database/sql
)

// SqlxDB возвращает подключение к бд.
// maxOpenConns ограничивает количество одновременно открытых соединений с бд,
// значение рассчитывается как размер пула воркеров + 2 стартовых запроса.
func SqlxDB(url string, maxOpenConns int) *sqlx.DB {
	db, err := sqlx.Connect("postgres", url)
	if err != nil {
		slog.New(slog.NewTextHandler(os.Stderr, nil)).Error(err.Error())
		os.Exit(1)
	}

	db.SetMaxOpenConns(maxOpenConns)

	return db
}
