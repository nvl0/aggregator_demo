package usecase

import (
	"aggregator/src/internal/entity/flow"
	"aggregator/src/internal/entity/global"

	"strings"

	"aggregator/src/rimport"

	"github.com/sirupsen/logrus"
)

type FlowUsecase struct {
	log *logrus.Logger
	//
	rimport.RepositoryImports
}

func NewFlowUsecase(
	log *logrus.Logger,
	ri rimport.RepositoryImports,
) *FlowUsecase {
	return &FlowUsecase{
		log:               log,
		RepositoryImports: ri,
	}
}

// PrepareFlow подготовка flow файла
func (u *FlowUsecase) PrepareFlow(dirName string) (flowStr string, err error) {
	lf := logrus.Fields{
		"dir_name": dirName,
	}

	// получение списка имен файлов с директории
	// чтобы перенести их в директорию ./tmp для считывания
	fileNameListInDir, err := u.Repository.Flow.ReadFileNamesInFlowDir(dirName)
	switch err {
	case nil:
		// перед тем как перенести flow необходимо убедиться
		// что flow файл имеет верный формат
		for _, fileName := range fileNameListInDir {
			if strings.Contains(fileName, flow.FlowNameSubStr) {
				// перенос flow файла в директорию ./tmp
				if err = u.Repository.Flow.MoveFlowToTempDir(dirName, fileName); err != nil {
					u.log.WithFields(lf).Errorln("не удалось переместить готовый flow в tmp, ошибка", err)
					return
				}
			}
		}

		// чтение flow файла с директории ./tmp
		if flowStr, err = u.Repository.Flow.ReadFlow(dirName); err != nil {
			u.log.WithFields(lf).Errorln("не удалось считать готовый flow с директории, ошибка", err)
			err = global.ErrInternalError
			return
		}

		return
	case global.ErrNoData:
		return
	default:
		u.log.WithFields(lf).Errorln("не удалось просмотреть директорию, ошибка", err)
		err = global.ErrInternalError
		return
	}
}
