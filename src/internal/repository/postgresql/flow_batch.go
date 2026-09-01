package postgresql

import (
	"errors"

	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/repository"
	"aggregator/src/internal/transaction"
	"aggregator/src/tools/gensql"

	"github.com/jmoiron/sqlx"
)

type flowBatchRepository struct {
}

func NewFlowBatchRepository() repository.FlowBatch {
	return &flowBatchRepository{}
}

// LoadCommittedFileNames загрузить имена закоммиченных flow файлов по nas_ip
func (r *flowBatchRepository) LoadCommittedFileNames(
	ts transaction.Session,
	nasIP string,
) (map[string]bool, error) {
	sqlQuery := `
		select fb.file_name
		from flow_batch fb
		where fb.nas_ip = $1`

	fileNameList, err := gensql.Select[string](SqlxTx(ts), sqlQuery, nasIP)
	switch {
	case err == nil:
	case errors.Is(err, global.ErrNoData):
		// чекпоинта по nas_ip нет, значит все файлы в tmp считаются новыми
		return map[string]bool{}, nil
	default:
		return nil, err
	}

	fileNameSet := make(map[string]bool, len(fileNameList))
	for _, fileName := range fileNameList {
		fileNameSet[fileName] = true
	}

	return fileNameSet, nil
}

// SaveFileNames сохранить имена flow файлов в чекпоинт
func (r *flowBatchRepository) SaveFileNames(
	ts transaction.Session,
	nasIP string,
	fileNameList []string,
) (err error) {
	if len(fileNameList) == 0 {
		return nil
	}

	var stmt *sqlx.Stmt

	// повторная запись уже известного имени не должна ломать транзакцию:
	// в чекпоинт пишется весь список файлов из tmp, включая закоммиченные ранее
	if stmt, err = SqlxTx(ts).Preparex(`
		insert into flow_batch (nas_ip, file_name)
		values ($1, $2)
		on conflict (nas_ip, file_name) do nothing
	`); err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	for _, fileName := range fileNameList {
		if _, err = stmt.Exec(nasIP, fileName); err != nil {
			return err
		}
	}

	return err
}

// RemoveByNasIP удалить чекпоинт по nas_ip
func (r *flowBatchRepository) RemoveByNasIP(ts transaction.Session, nasIP string) error {
	_, err := SqlxTx(ts).Exec(`
		delete from flow_batch
		where nas_ip = $1`, nasIP)

	return err
}
