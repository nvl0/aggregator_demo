package postgresql

import (
	"aggregator/app/internal/entity/session"
	"aggregator/app/internal/repository"
	"aggregator/app/internal/transaction"
	"aggregator/app/tools/gensql"

	"github.com/jmoiron/sqlx"
)

type sessionRepository struct {
}

func NewSessionRepository() repository.Session {
	return &sessionRepository{}
}

// LoadOnlineSessionList загрузить онлайн сессий из таблицы
func (r *sessionRepository) LoadOnlineSessionList(ts transaction.Session) ([]session.Session, error) {
	sqlQuery := `
		select s.ip, s.sess_id, s.nas_ip
		from session s`

	return gensql.Select[session.Session](SqlxTx(ts), sqlQuery)
}

// SaveChunkList сохранить чанки по клиентской сессии
func (r *sessionRepository) SaveChunkList(ts transaction.Session, chunkList []session.Chunk) (err error) {
	var stmt *sqlx.NamedStmt

	if stmt, err = SqlxTx(ts).PrepareNamed(`
		insert into chunk (sess_id, channel_id, upload, download)
		values (:sess_id, :channel_id, :upload, :download)
	`); err != nil {
		return
	}
	defer stmt.Close()

	for _, chunk := range chunkList {
		if _, err = stmt.Exec(&chunk); err != nil {
			return
		}
	}

	return
}
