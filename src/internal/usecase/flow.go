package usecase

import (
	"errors"
	"log/slog"
	"strings"

	"aggregator/src/internal/entity/flow"
	"aggregator/src/internal/entity/global"
	"aggregator/src/rimport"
)

type FlowUsecase struct {
	log *slog.Logger
	//
	rimport.RepositoryImports
}

func NewFlowUsecase(
	log *slog.Logger,
	ri rimport.RepositoryImports,
) *FlowUsecase {
	return &FlowUsecase{
		log:               log,
		RepositoryImports: ri,
	}
}

// StreamFlow подготовка и потоковое чтение flow.
// skipFileNames — имена файлов, чанки которых уже закоммичены: их содержимое
// в onLine не попадает, но имена возвращаются в fileNameList
func (u *FlowUsecase) StreamFlow(
	dirName string,
	skipFileNames map[string]bool,
	onLine func(line string) error,
) (fileNameList []string, flowSize int, err error) {
	// получение списка имен файлов с директории
	// чтобы перенести их в директорию ./tmp для считывания
	fileNameListInDir, err := u.Repository.Flow.ReadFileNamesInFlowDir(dirName)
	switch {
	case err == nil:
		// перед тем как перенести flow необходимо убедиться
		// что flow файл имеет верный формат
		for _, fileName := range fileNameListInDir {
			if strings.Contains(fileName, flow.FlowNameSubStr) {
				// перенос flow файла в директорию ./tmp
				if err = u.Repository.Flow.MoveFlowToTempDir(dirName, fileName); err != nil {
					u.log.Error("не удалось переместить готовый flow в tmp, ошибка",
						"error", err, "dir_name", dirName)
					return fileNameList, flowSize, err
				}
			}
		}

		// потоковое чтение flow файлов с директории ./tmp
		if fileNameList, flowSize, err = u.Repository.Flow.StreamFlow(
			dirName, skipFileNames, onLine); err != nil {
			u.log.Error("не удалось считать готовый flow с директории, ошибка",
				"error", err, "dir_name", dirName)
			err = global.ErrInternalError
			return fileNameList, flowSize, err
		}

		return fileNameList, flowSize, err
	case errors.Is(err, global.ErrNoData):
		return fileNameList, flowSize, err
	default:
		u.log.Error("не удалось просмотреть директорию, ошибка", "error", err, "dir_name", dirName)
		err = global.ErrInternalError
		return fileNameList, flowSize, err
	}
}
