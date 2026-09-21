package pgdb_test

import (
	"os"
	"testing"

	"aggregator/src/config"
	"aggregator/src/tools/pgdb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSqlxDBConnects проверяет, что SqlxDB возвращает рабочее подключение
// с примененным лимитом maxOpenConns.
func TestSqlxDBConnects(t *testing.T) {
	conf, err := config.NewConfig(os.Getenv("CONF_PATH"))
	require.NoError(t, err)
	require.NotEmpty(t, conf)

	const maxOpenConns = 5

	db := pgdb.SqlxDB(conf.PostgresURL(), maxOpenConns)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, db.Ping())
	assert.Equal(t, maxOpenConns, db.Stats().MaxOpenConnections)
}
