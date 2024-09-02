package transaction

type Session interface {
	Start() error
	Rollback() error
	Commit() error
	Tx() interface{}
	TxIsActive() bool
	CreateNewSession() Session
}

type SessionManager interface {
	CreateSession() Session
}
