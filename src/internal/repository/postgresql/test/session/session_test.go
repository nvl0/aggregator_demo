package session_test

import (
	"aggregator/src/config"
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/repository/postgresql"
	"aggregator/src/internal/transaction"
	"aggregator/src/tools/gensql"
	"aggregator/src/tools/pgdb"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadOnlineSessionList(t *testing.T) {
	r := require.New(t)

	conf, err := config.NewConfig(os.Getenv("CONF_PATH"))
	r.NoError(err)
	r.NotEmpty(conf)

	db := pgdb.SqlxDB(conf.PostgresURL())
	r.NoError(db.Ping())

	repo := postgresql.NewSessionRepository()

	ts := transaction.NewSQLSession(db)
	r.NoError(ts.Start())
	defer ts.Rollback()

	t.Run("подготовка данных", func(t *testing.T) {
		expectedData := session.OnlineSession{
			SessID: 0,
			IP:     "127.0.0.2",
			NasIP:  "127.0.0.0",
		}

		_, err = postgresql.SqlxTx(ts).NamedExec(`
			insert into online_session (ip, sess_id, nas_ip)
			values (:ip, :sess_id, :nas_ip)
		`, expectedData)
		r.NoError(err)

		t.Run("проверка данных", func(t *testing.T) {
			data, err := repo.LoadOnlineSessionList(ts)
			r.NoError(err)
			r.Contains(data, expectedData)
		})
	})
}

func TestSaveChunkList(t *testing.T) {
	r := require.New(t)

	conf, err := config.NewConfig(os.Getenv("CONF_PATH"))
	r.NoError(err)
	r.NotEmpty(conf)

	db := pgdb.SqlxDB(conf.PostgresURL())
	r.NoError(db.Ping())

	repo := postgresql.NewSessionRepository()

	ts := transaction.NewSQLSession(db)
	r.NoError(ts.Start())
	defer ts.Rollback()

	t.Run("сохранение данных", func(t *testing.T) {
		chunk := session.Chunk{
			SessID:    1,
			ChannelID: 1,
			Download:  63543,
			Upload:    4234,
		}

		expectedData := []session.Chunk{chunk, chunk}

		r.NoError(repo.SaveChunkList(ts, expectedData))

		t.Run("проверка данных", func(t *testing.T) {
			data, err := gensql.Select[session.Chunk](postgresql.SqlxTx(ts), `
				select sess_id, channel_id, download, upload
				from chunk
				where sess_id = $1 and channel_id = $2
			`, chunk.SessID, chunk.ChannelID)
			r.NoError(err)
			r.Equal(data, expectedData)
		})
	})
}
