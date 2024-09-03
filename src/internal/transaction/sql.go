package transaction

import (
	"github.com/jmoiron/sqlx"
)

type sqlSession struct {
	db        *sqlx.DB
	init      bool
	currentTx *sqlx.Tx
}

func NewSQLSession(db *sqlx.DB) Session {
	return &sqlSession{db: db}
}

func (t *sqlSession) Start() (err error) {
	if t.init && t.currentTx != nil {
		err = ErrActiveTransaction
		return
	}
	t.init = true
	t.currentTx, err = t.db.Beginx()
	return
}

func (t *sqlSession) Rollback() (err error) {
	switch {
	case !t.init:
		err = ErrNotInit
	case t.currentTx == nil:
		err = ErrClosed
	default:
		err = t.currentTx.Rollback()
		t.init = false
		t.currentTx = nil
	}

	return
}

func (t *sqlSession) Commit() (err error) {
	switch {
	case !t.init:
		err = ErrNotInit
	case t.currentTx == nil:
		err = ErrClosed
	default:
		err = t.currentTx.Commit()
	}

	return
}

func (t *sqlSession) Tx() interface{} {
	return t.currentTx
}

func (t *sqlSession) TxIsActive() bool {
	return t.init && t.currentTx != nil
}

type sqlSessionManager struct {
	db *sqlx.DB
}

func NewSQLSessionManager(db *sqlx.DB) SessionManager {
	return &sqlSessionManager{db: db}
}

func (s *sqlSessionManager) CreateSession() Session {
	return NewSQLSession(s.db)
}
