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

// PrepareFlow подготовка flow файла.
// skipFileNames — имена файлов, чанки которых уже закоммичены: их содержимое
// в flowStr не попадает, но имена возвращаются в fileNameList.
func (u *FlowUsecase) PrepareFlow(
	dirName string,
	skipFileNames map[string]bool,
) (flowStr string, fileNameList []string, err error) {
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
					u.log.Error("не удалось переместить готовый flow в tmp, ошибка", "error", err, "dir_name", dirName)
					return flowStr, fileNameList, err
				}
			}
		}

		// чтение flow файла с директории ./tmp
		if flowStr, fileNameList, err = u.Repository.Flow.ReadFlow(dirName, skipFileNames); err != nil {
			u.log.Error("не удалось считать готовый flow с директории, ошибка", "error", err, "dir_name", dirName)
			err = global.ErrInternalError
			return flowStr, fileNameList, err
		}

		return flowStr, fileNameList, err
	case errors.Is(err, global.ErrNoData):
		return flowStr, fileNameList, err
	default:
		u.log.Error("не удалось просмотреть директорию, ошибка", "error", err, "dir_name", dirName)
		err = global.ErrInternalError
		return flowStr, fileNameList, err
	}
}
