package pgdb

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// SqlxDB get db link
func SqlxDB(URL string) *sqlx.DB {
	db, err := sqlx.Connect("postgres", URL)
	if err != nil {
		log.Fatalln(err)
	}
	return db
}
