package gensql

import (
	"database/sql"
	"errors"

	"aggregator/src/internal/entity/global"

	"github.com/jmoiron/sqlx"
)

func Get[T any](tx *sqlx.Tx, sqlQuery string, params ...interface{}) (t T, err error) {
	var data T

	err = tx.Get(&data, sqlQuery, params...)

	switch {
	case err == nil:
		return data, nil
	case errors.Is(err, sql.ErrNoRows):
		err = global.ErrNoData
		return t, err
	default:
		return t, err
	}
}

func GetNamed[T any](tx *sqlx.Tx, sqlQuery string, params map[string]interface{}) (t T, err error) {
	var data T

	stmt, err := tx.PrepareNamed(sqlQuery)
	if err != nil {
		return t, err
	}
	defer stmt.Close()

	err = stmt.Get(&data, params)
	switch {
	case err == nil:
		return data, nil
	case errors.Is(err, sql.ErrNoRows):
		err = global.ErrNoData
		return t, err
	default:
		return t, err
	}
}

func GetNamedStruct[T any, S any](tx *sqlx.Tx, sqlQuery string, s S) (t T, err error) {
	var data T

	stmt, err := tx.PrepareNamed(sqlQuery)
	if err != nil {
		return t, err
	}
	defer stmt.Close()

	err = stmt.Get(&data, s)
	switch {
	case err == nil:
		return data, nil
	case errors.Is(err, sql.ErrNoRows):
		err = global.ErrNoData
		return t, err
	default:
		return t, err
	}
}
