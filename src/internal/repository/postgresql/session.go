package postgresql

import (
	"context"

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
func (r *sessionRepository) LoadOnlineSessionList(
	ctx context.Context,
	ts transaction.Session,
) ([]session.OnlineSession, error) {
	sqlQuery := `
		select o.ip, o.sess_id, o.nas_ip
		from online_session o`

	return gensql.Select[session.OnlineSession](ctx, ts.Tx(), sqlQuery)
}

// SaveChunkList сохранить чанки по клиентской сессии
func (r *sessionRepository) SaveChunkList(
	ctx context.Context,
	ts transaction.Session,
	chunkList []session.Chunk,
) (err error) {
	var stmt *sqlx.NamedStmt

	if stmt, err = ts.Tx().PrepareNamedContext(ctx, `
		insert into chunk (sess_id, channel_id, upload, download)
		values (:sess_id, :channel_id, :upload, :download)
	`); err != nil {
		return err
	}
	defer stmt.Close()

	for _, chunk := range chunkList {
		if _, err = stmt.ExecContext(ctx, &chunk); err != nil {
			return err
		}
	}

	return err
}
