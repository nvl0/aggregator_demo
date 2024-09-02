package usecase

import (
	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/transaction"
	"aggregator/src/rimport"
	"fmt"

	"github.com/sirupsen/logrus"
)

type SessionUsecase struct {
	log *logrus.Logger
	//
	rimport.RepositoryImports
}

func NewSessionUsecase(
	log *logrus.Logger,
	ri rimport.RepositoryImports,
) *SessionUsecase {
	return &SessionUsecase{
		log:               log,
		RepositoryImports: ri,
	}
}

func (u *SessionUsecase) logPrefix() string {
	return "[session_usecase]"
}

// LoadOnlineSessionListByNasIP получение списка онлайн сессий и сортировка по nas_ip
func (u *SessionUsecase) LoadOnlineSessionListByNasIP(ts transaction.Session) (
	sessionMap map[string][]session.Session, err error) {
	sessionMap = make(map[string][]session.Session)

	// получение списка онлайн сессий
	sessionList, err := u.Repository.Session.LoadOnlineSessionList(ts)
	switch err {
	case nil:

		// сортировка по nas_ip
		for _, sess := range sessionList {
			sessionMap[sess.NasIP] = append(sessionMap[sess.NasIP], sess)
		}

		return
	case global.ErrNoData:
		u.log.Errorln(
			u.logPrefix(),
			fmt.Sprintf("не удалось загрузить список онлайн сессий; ошибка: %v", err),
		)
		return
	default:
		u.log.Errorln(
			u.logPrefix(),
			fmt.Sprintf("не удалось загрузить список онлайн сессий; ошибка: %v", err),
		)
		err = global.ErrInternalError
		return
	}
}
