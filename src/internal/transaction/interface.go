package transaction

type Session interface {
	Start() error
	Rollback() error
	Commit() error
	Tx() interface{}
	TxIsActive() bool
}

type SessionManager interface {
	CreateSession() Session
}
