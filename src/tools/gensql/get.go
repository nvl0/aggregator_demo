package gensql

import (
	"aggregator/src/internal/entity/global"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

func Get[T any](tx *sqlx.Tx, sqlQuery string, params ...interface{}) (t T, err error) {
	var data T

	err = tx.Get(&data, sqlQuery, params...)

	switch err {
	case nil:
		return data, nil
	case sql.ErrNoRows:
		err = global.ErrNoData
		return
	default:
		return
	}
}

func GetNamed[T any](tx *sqlx.Tx, sqlQuery string, params map[string]interface{}) (t T, err error) {
	var data T

	stmt, err := tx.PrepareNamed(sqlQuery)
	if err != nil {
		return
	}
	defer stmt.Close()

	err = stmt.Get(&data, params)
	switch err {
	case nil:
		return data, nil
	case sql.ErrNoRows:
		err = global.ErrNoData
		return
	default:
		return
	}
}

func GetNamedStruct[T any, S any](tx *sqlx.Tx, sqlQuery string, s S) (t T, err error) {
	var data T

	stmt, err := tx.PrepareNamed(sqlQuery)
	if err != nil {
		return
	}
	defer stmt.Close()

	err = stmt.Get(&data, s)
	switch err {
	case nil:
		return data, nil
	case sql.ErrNoRows:
		err = global.ErrNoData
		return
	default:
		return
	}
}
