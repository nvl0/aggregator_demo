package flowbatch_test

import (
	"aggregator/src/config"
	"aggregator/src/internal/repository/postgresql"
	"aggregator/src/internal/transaction"
	"aggregator/src/tools/pgdb"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFlowBatch чекпоинт закоммиченных flow файлов: чтение, запись, повторная запись, удаление
func TestFlowBatch(t *testing.T) {
	r := require.New(t)

	const (
		nasIP      = "127.0.0.0"
		otherNasIP = "192.168.0.0"
		fileName1  = "ft-01.01.2026-00:00:00"
		fileName2  = "ft-01.01.2026-00:05:00"
	)

	conf, err := config.NewConfig(os.Getenv("CONF_PATH"))
	r.NoError(err)
	r.NotEmpty(conf)

	db := pgdb.SqlxDB(conf.PostgresURL(), conf.WorkerPoolSize()+2)
	r.NoError(db.Ping())

	repo := postgresql.NewFlowBatchRepository()

	// вся проверка идет внутри одной транзакции, которая откатывается в конце,
	// поэтому чистить таблицу отдельно не нужно
	ts := transaction.NewSQLSession(db)
	r.NoError(ts.Start())
	defer ts.Rollback()

	t.Run("пустой чекпоинт не является ошибкой", func(_ *testing.T) {
		data, errLoad := repo.LoadCommittedFileNames(ts, nasIP)
		r.NoError(errLoad)
		r.Empty(data)
	})

	t.Run("сохранение имен", func(t *testing.T) {
		r.NoError(repo.SaveFileNames(ts, nasIP, []string{fileName1, fileName2}))
		r.NoError(repo.SaveFileNames(ts, otherNasIP, []string{fileName1}))

		t.Run("проверка данных", func(_ *testing.T) {
			data, errLoad := repo.LoadCommittedFileNames(ts, nasIP)
			r.NoError(errLoad)
			r.Equal(map[string]bool{fileName1: true, fileName2: true}, data)
		})

		t.Run("повторное сохранение тех же имен не является ошибкой", func(_ *testing.T) {
			r.NoError(repo.SaveFileNames(ts, nasIP, []string{fileName1, fileName2}))

			data, errLoad := repo.LoadCommittedFileNames(ts, nasIP)
			r.NoError(errLoad)
			r.Len(data, 2)
		})

		t.Run("пустой список имен не является ошибкой", func(_ *testing.T) {
			r.NoError(repo.SaveFileNames(ts, nasIP, nil))
		})
	})

	t.Run("удаление по nas_ip", func(t *testing.T) {
		r.NoError(repo.RemoveByNasIP(ts, nasIP))

		t.Run("проверка данных", func(_ *testing.T) {
			data, errLoad := repo.LoadCommittedFileNames(ts, nasIP)
			r.NoError(errLoad)
			r.Empty(data)

			// записи других nas_ip удаление не затрагивает
			otherData, errOther := repo.LoadCommittedFileNames(ts, otherNasIP)
			r.NoError(errOther)
			r.Equal(map[string]bool{fileName1: true}, otherData)
		})
	})
}
