package gensql_test

import (
	"context"
	"os"
	"testing"

	"aggregator/src/config"
	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/transaction"
	"aggregator/src/tools/gensql"
	"aggregator/src/tools/pgdb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type row struct {
	ID int `db:"id"`
}

func newSession(t *testing.T) transaction.Session {
	t.Helper()

	conf, err := config.NewConfig(os.Getenv("CONF_PATH"))
	require.NoError(t, err)
	require.NotEmpty(t, conf)

	db := pgdb.SqlxDB(conf.PostgresURL(), conf.WorkerPoolSize()+2)
	require.NoError(t, db.Ping())

	ts := transaction.NewSQLSession(db)
	require.NoError(t, ts.Start(context.Background()))
	t.Cleanup(func() { _ = ts.Rollback() })

	return ts
}

func TestGet(t *testing.T) {
	ctx := context.Background()

	t.Run("одна строка находится", func(t *testing.T) {
		ts := newSession(t)

		data, err := gensql.Get[row](ctx, ts.Tx(), "select 1 as id")
		require.NoError(t, err)
		assert.Equal(t, 1, data.ID)
	})

	t.Run("нет строк - ErrNoData", func(t *testing.T) {
		ts := newSession(t)

		_, err := gensql.Get[row](ctx, ts.Tx(), "select 1 as id where false")
		assert.ErrorIs(t, err, global.ErrNoData)
	})

	t.Run("ошибка sql пробрасывается как есть", func(t *testing.T) {
		ts := newSession(t)

		_, err := gensql.Get[row](ctx, ts.Tx(), "select * from no_such_table_xyz")
		require.Error(t, err)
		assert.NotErrorIs(t, err, global.ErrNoData)
	})
}

func TestGetNamed(t *testing.T) {
	ctx := context.Background()

	t.Run("именованный параметр подставляется", func(t *testing.T) {
		ts := newSession(t)

		data, err := gensql.GetNamed[row](
			ctx, ts.Tx(), "select cast(:val as int) as id", map[string]any{"val": 7},
		)
		require.NoError(t, err)
		assert.Equal(t, 7, data.ID)
	})

	t.Run("нет строк - ErrNoData", func(t *testing.T) {
		ts := newSession(t)

		_, err := gensql.GetNamed[row](
			ctx, ts.Tx(), "select cast(:val as int) as id where false", map[string]any{"val": 7},
		)
		assert.ErrorIs(t, err, global.ErrNoData)
	})
}

type valParam struct {
	Val int `db:"val"`
}

func TestGetNamedStruct(t *testing.T) {
	ctx := context.Background()

	ts := newSession(t)

	data, err := gensql.GetNamedStruct[row](ctx, ts.Tx(), "select cast(:val as int) as id", valParam{Val: 9})
	require.NoError(t, err)
	assert.Equal(t, 9, data.ID)
}

func TestSelect(t *testing.T) {
	ctx := context.Background()

	t.Run("несколько строк", func(t *testing.T) {
		ts := newSession(t)

		data, err := gensql.Select[row](ctx, ts.Tx(), "select generate_series(1,3) as id")
		require.NoError(t, err)
		assert.Len(t, data, 3)
	})

	t.Run("нет строк - ErrNoData", func(t *testing.T) {
		ts := newSession(t)

		_, err := gensql.Select[row](ctx, ts.Tx(), "select 1 as id where false")
		assert.ErrorIs(t, err, global.ErrNoData)
	})

	t.Run("ошибка sql пробрасывается как есть", func(t *testing.T) {
		ts := newSession(t)

		_, err := gensql.Select[row](ctx, ts.Tx(), "select * from no_such_table_xyz")
		require.Error(t, err)
		assert.NotErrorIs(t, err, global.ErrNoData)
	})
}

func TestSelectNamed(t *testing.T) {
	ctx := context.Background()

	t.Run("именованный параметр подставляется", func(t *testing.T) {
		ts := newSession(t)

		data, err := gensql.SelectNamed[row](
			ctx, ts.Tx(), "select cast(:val as int) as id", map[string]any{"val": 4},
		)
		require.NoError(t, err)
		require.Len(t, data, 1)
		assert.Equal(t, 4, data[0].ID)
	})

	t.Run("нет строк - ErrNoData", func(t *testing.T) {
		ts := newSession(t)

		_, err := gensql.SelectNamed[row](
			ctx, ts.Tx(), "select cast(:val as int) as id where false", map[string]any{"val": 4},
		)
		assert.ErrorIs(t, err, global.ErrNoData)
	})
}
