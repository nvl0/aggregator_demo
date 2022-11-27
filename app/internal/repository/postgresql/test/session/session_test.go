package session_test

import (
	"aggregator/app/config"
	"aggregator/app/internal/entity/global"
	"aggregator/app/internal/entity/session"
	"aggregator/app/internal/repository/postgresql"
	"aggregator/app/internal/transaction"
	"aggregator/app/tools/gensql"
	"aggregator/app/tools/pgdb"
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
	ts.Start()
	defer ts.Rollback()

	t.Run("подготовка данных", func(t *testing.T) {

		sess := session.Session{
			SessID: 1,
			IP:     "127.0.0.2",
			NasIP:  "127.0.0.0",
		}

		_, err = postgresql.SqlxTx(ts).NamedExec(`
			insert into session (ip, sess_id, nas_ip)
			values (:ip, :sess_id, :nas_ip)
		`, sess)
		r.NoError(err)

		t.Run("проверка данных", func(t *testing.T) {
			data, err := repo.LoadOnlineSessionList(ts)
			r.NoError(err)

			expectedData := session.Session{
				SessID: sess.SessID,
				IP:     sess.IP,
				NasIP:  sess.NasIP,
			}

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
	ts.Start()
	defer ts.Rollback()

	data := session.Chunk{
		SessID:    1,
		ChannelID: int(global.Internet),
		Download:  63543,
		Upload:    4234,
	}

	chunkList := []session.Chunk{data, data}

	t.Run("сохранение данных", func(t *testing.T) {
		r.NoError(repo.SaveChunkList(ts, chunkList))

		t.Run("проверка данных", func(t *testing.T) {
			expectedData, err := gensql.Select[session.Chunk](postgresql.SqlxTx(ts), `
				select sess_id, channel_id, download, upload
				from chunk
				where sess_id = $1 and channel_id = $2
			`, data.SessID, data.ChannelID)
			r.NoError(err)

			r.Equal(expectedData, chunkList)
		})
	})
}
