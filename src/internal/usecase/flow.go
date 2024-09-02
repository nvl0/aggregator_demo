package usecase

import (
	"aggregator/src/internal/entity/flow"
	"aggregator/src/internal/entity/global"

	"strings"

	"aggregator/src/rimport"
	"fmt"

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

func (u *FlowUsecase) logPrefix() string {
	return "[flow_usecase]"
}

// PrepareFlow подготовка с считывание готового flow файла
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
					u.log.WithFields(lf).Errorln(u.logPrefix(),
						fmt.Sprintf("не удалось переместить готовый flow в tmp; ошибка: %v", err))
					return
				}
			}
		}

		// чтение flow файла с директории ./tmp
		flowStr, err = u.Repository.Flow.ReadFlow(dirName)
		if err != nil {
			u.log.WithFields(lf).Errorln(
				u.logPrefix(),
				fmt.Sprintf("не удалось считать готовый flow с директории; ошибка: %v", err),
			)
			err = global.ErrInternalError
			return
		}

		return
	case global.ErrNoData:
		u.log.WithFields(lf).Debugln(
			u.logPrefix(),
			fmt.Sprintf("не удалось просмотреть директорию; ошибка: %v", err),
		)
		return
	default:
		u.log.WithFields(lf).Errorln(
			u.logPrefix(),
			fmt.Sprintf("не удалось просмотреть директорию; ошибка: %v", err),
		)
		err = global.ErrInternalError
		return
	}
}
