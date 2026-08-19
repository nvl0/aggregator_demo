package postgresql

import (
	"aggregator/src/internal/transaction"

	"github.com/jmoiron/sqlx"
)

func SqlxTx(ts transaction.Session) *sqlx.Tx {
	tx, ok := ts.Tx().(*sqlx.Tx)
	if !ok {
		panic("transaction.Session.Tx() вернул не *sqlx.Tx")
	}
	return tx
}
