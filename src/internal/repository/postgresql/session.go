package postgresql

import (
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/repository"
	"aggregator/src/internal/transaction"
	"aggregator/src/tools/gensql"

	"github.com/jmoiron/sqlx"
)

type sessionRepository struct {
}

func NewSessionRepository() repository.Session {
	return &sessionRepository{}
}

// LoadOnlineSessionList загрузить онлайн сессий из таблицы
func (r *sessionRepository) LoadOnlineSessionList(ts transaction.Session) ([]session.OnlineSession, error) {
	sqlQuery := `
		select o.ip, o.sess_id, o.nas_ip
		from online_session o`

	return gensql.Select[session.OnlineSession](SqlxTx(ts), sqlQuery)
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
