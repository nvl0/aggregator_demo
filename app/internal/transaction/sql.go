package transaction

import (
	"log"

	"github.com/jmoiron/sqlx"
)

type sqlSession struct {
	db *sqlx.DB
	//
	currentTx *sqlx.Tx
}

func NewSQLSession(db *sqlx.DB) Session {
	return &sqlSession{db: db}
}

func (t *sqlSession) Start() (err error) {
	if t.currentTx != nil {
		log.Fatalln("открытие транзакции при активной транзакции")
	}
	t.currentTx, err = t.db.Beginx()
	return
}

func (t *sqlSession) Rollback() error {
	err := t.currentTx.Rollback()
	t.currentTx = nil
	return err
}

func (t *sqlSession) Commit() error {
	err := t.currentTx.Commit()
	return err
}

func (t *sqlSession) Tx() interface{} {
	return t.currentTx
}

func (t *sqlSession) TxIsActive() bool {
	return t.currentTx != nil
}

func (t *sqlSession) CreateNewSession() Session {
	return NewSQLSession(t.db)
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
