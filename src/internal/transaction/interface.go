package transaction

import "context"

type Session interface {
	Start(ctx context.Context) error
	Rollback() error
	Commit() error
	Tx() interface{}
	TxIsActive() bool
}

type SessionManager interface {
	CreateSession() Session
}
