package gensql

import (
	"context"
	"database/sql"
	"errors"

	"aggregator/src/internal/entity/global"

	"github.com/jmoiron/sqlx"
)

func Select[T any](ctx context.Context, tx *sqlx.Tx, sqlQuery string, params ...interface{}) ([]T, error) {
	data := make([]T, 0)

	err := tx.SelectContext(ctx, &data, sqlQuery, params...)

	if err == nil && len(data) == 0 {
		err = sql.ErrNoRows
	}

	switch {
	case err == nil:
		return data, nil
	case errors.Is(err, sql.ErrNoRows):
		return nil, global.ErrNoData
	default:
		return nil, err
	}
}

func SelectNamed[T any](ctx context.Context, tx *sqlx.Tx, sqlQuery string, params map[string]interface{}) ([]T, error) {
	data := make([]T, 0)

	stmt, err := tx.PrepareNamedContext(ctx, sqlQuery)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	err = stmt.SelectContext(ctx, &data, params)

	if err == nil && len(data) == 0 {
		err = sql.ErrNoRows
	}

	switch {
	case err == nil:
		return data, nil
	case errors.Is(err, sql.ErrNoRows):
		return nil, global.ErrNoData
	default:
		return nil, err
	}
}
