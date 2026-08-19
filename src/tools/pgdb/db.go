package pgdb

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// SqlxDB возвращает подключение к бд.
// maxOpenConns ограничивает количество одновременно открытых соединений с бд,
// значение рассчитывается как размер пула воркеров + 2 стартовых запроса.
func SqlxDB(URL string, maxOpenConns int) *sqlx.DB {
	db, err := sqlx.Connect("postgres", URL)
	if err != nil {
		log.Fatalln(err)
	}

	db.SetMaxOpenConns(maxOpenConns)

	return db
}
