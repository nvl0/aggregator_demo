package session_test

import (
	"context"
	"os"
	"testing"

	"aggregator/src/config"
	"aggregator/src/internal/transaction"
	"aggregator/src/tools/pgdb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSession(t *testing.T) transaction.Session {
	t.Helper()

	conf, err := config.NewConfig(os.Getenv("CONF_PATH"))
	require.NoError(t, err)
	require.NotEmpty(t, conf)

	db := pgdb.SqlxDB(conf.PostgresURL(), conf.WorkerPoolSize()+2)
	require.NoError(t, db.Ping())

	return transaction.NewSQLSession(db)
}

// TestSessionStartTwice повторный Start при уже активной транзакции -> ErrActiveTransaction
func TestSessionStartTwice(t *testing.T) {
	ts := newSession(t)
	ctx := context.Background()

	require.NoError(t, ts.Start(ctx))
	defer func() { _ = ts.Rollback() }()

	err := ts.Start(ctx)
	assert.ErrorIs(t, err, transaction.ErrActiveTransaction)
}

// TestSessionRollbackWithoutStart Rollback без предварительного Start -> ErrNotInit
func TestSessionRollbackWithoutStart(t *testing.T) {
	ts := newSession(t)

	err := ts.Rollback()
	assert.ErrorIs(t, err, transaction.ErrNotInit)
}

// TestSessionCommitWithoutStart Commit без предварительного Start -> ErrNotInit
func TestSessionCommitWithoutStart(t *testing.T) {
	ts := newSession(t)

	err := ts.Commit()
	assert.ErrorIs(t, err, transaction.ErrNotInit)
}

// TestSessionRollbackAfterCommit повторный Rollback после Commit -> ErrNotInit.
// ErrClosed объявлен в пакете, но недостижим через публичный API sqlSession:
// Commit/Rollback всегда сбрасывают init и currentTx вместе, поэтому условие
// "currentTx == nil при init == true" никогда не выполняется.
func TestSessionRollbackAfterCommit(t *testing.T) {
	ts := newSession(t)
	ctx := context.Background()

	require.NoError(t, ts.Start(ctx))
	require.NoError(t, ts.Commit())

	err := ts.Rollback()
	assert.ErrorIs(t, err, transaction.ErrNotInit)
}

// TestSessionCommitAfterCommit повторный Commit после Commit -> ErrNotInit (см. TestSessionRollbackAfterCommit)
func TestSessionCommitAfterCommit(t *testing.T) {
	ts := newSession(t)
	ctx := context.Background()

	require.NoError(t, ts.Start(ctx))
	require.NoError(t, ts.Commit())

	err := ts.Commit()
	assert.ErrorIs(t, err, transaction.ErrNotInit)
}

// TestSessionTxIsActiveTransitions TxIsActive переключается false -> true -> false по циклу Start/Commit
func TestSessionTxIsActiveTransitions(t *testing.T) {
	ts := newSession(t)
	ctx := context.Background()

	assert.False(t, ts.TxIsActive(), "до Start транзакция не активна")

	require.NoError(t, ts.Start(ctx))
	assert.True(t, ts.TxIsActive(), "после Start транзакция активна")

	require.NoError(t, ts.Commit())
	assert.False(t, ts.TxIsActive(), "после Commit транзакция не активна")
}

// TestSessionTx Tx() возвращает nil до Start и рабочий *sqlx.Tx после
func TestSessionTx(t *testing.T) {
	ts := newSession(t)
	ctx := context.Background()

	assert.Nil(t, ts.Tx(), "до Start Tx() должен быть nil")

	require.NoError(t, ts.Start(ctx))
	defer func() { _ = ts.Rollback() }()

	tx := ts.Tx()
	require.NotNil(t, tx)
	assert.NoError(t, tx.QueryRowContext(ctx, "select 1").Err())
}
