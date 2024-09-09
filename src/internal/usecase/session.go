package usecase

import (
	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/transaction"
	"aggregator/src/rimport"

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

// LoadOnlineSessionMap map[nas_ip][]session.OnlineSession
func (u *SessionUsecase) LoadOnlineSessionMap(ts transaction.Session) (
	sessionMap map[string][]session.OnlineSession, err error) {

	// получение списка онлайн сессий
	sessionList, err := u.Repository.Session.LoadOnlineSessionList(ts)
	switch err {
	case nil:
		sessionMap = make(map[string][]session.OnlineSession)

		// сортировка по nas_ip
		for _, sess := range sessionList {
			sessionMap[sess.NasIP] = append(sessionMap[sess.NasIP], sess)
		}

		return
	case global.ErrNoData:
		return
	default:
		u.log.Errorln("не удалось загрузить список онлайн сессий, ошибка", err)
		err = global.ErrInternalError
		return
	}
}
