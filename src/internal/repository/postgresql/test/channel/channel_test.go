package channel_test

import (
	"aggregator/src/config"
	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/repository/postgresql"
	"aggregator/src/internal/transaction"
	"aggregator/src/tools/gensql"
	"aggregator/src/tools/pgdb"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadChannelList(t *testing.T) {
	r := require.New(t)

	conf, err := config.NewConfig(os.Getenv("CONF_PATH"))
	r.NoError(err)
	r.NotEmpty(conf)

	db := pgdb.SqlxDB(conf.PostgresURL())
	r.NoError(db.Ping())

	repo := postgresql.NewChannelRepository()

	ts := transaction.NewSQLSession(db)
	r.NoError(ts.Start())
	defer ts.Rollback()

	t.Run("подготовка данных", func(t *testing.T) {
		expectedData := channel.Channel{
			Enabled: true,
			Descr:   "repo_test",
		}

		expectedData.ID, err = gensql.GetNamedStruct[int](postgresql.SqlxTx(ts), `
			insert into channel (enabled, descr)
			values (:enabled, :descr)
			returning channel_id
		`, expectedData)
		r.NoError(err)

		t.Run("проверка данных", func(t *testing.T) {
			data, err := repo.LoadChannelList(ts)
			r.NoError(err)
			r.Contains(data, expectedData)
		})
	})
}
