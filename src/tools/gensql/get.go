package gensql

import (
	"context"
	"database/sql"
	"errors"

	"aggregator/src/internal/entity/global"

	"github.com/jmoiron/sqlx"
)

func Get[T any](ctx context.Context, tx *sqlx.Tx, sqlQuery string, params ...interface{}) (t T, err error) {
	var data T

	err = tx.GetContext(ctx, &data, sqlQuery, params...)

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

func GetNamed[T any](
	ctx context.Context,
	tx *sqlx.Tx,
	sqlQuery string,
	params map[string]interface{},
) (t T, err error) {
	var data T

	stmt, err := tx.PrepareNamedContext(ctx, sqlQuery)
	if err != nil {
		return t, err
	}
	defer stmt.Close()

	err = stmt.GetContext(ctx, &data, params)
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

func GetNamedStruct[T any, S any](ctx context.Context, tx *sqlx.Tx, sqlQuery string, s S) (t T, err error) {
	var data T

	stmt, err := tx.PrepareNamedContext(ctx, sqlQuery)
	if err != nil {
		return t, err
	}
	defer stmt.Close()

	err = stmt.GetContext(ctx, &data, s)
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
