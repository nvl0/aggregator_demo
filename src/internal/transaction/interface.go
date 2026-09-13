package transaction

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type Session interface {
	Start(ctx context.Context) error
	Rollback() error
	Commit() error
	Tx() *sqlx.Tx
	TxIsActive() bool
}

type SessionManager interface {
	CreateSession() Session
}
