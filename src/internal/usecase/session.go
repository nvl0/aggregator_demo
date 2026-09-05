package usecase

import (
	"errors"
	"log/slog"

	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/transaction"
	"aggregator/src/rimport"
)

type SessionUsecase struct {
	log *slog.Logger
	//
	rimport.RepositoryImports
}

func NewSessionUsecase(
	log *slog.Logger,
	ri rimport.RepositoryImports,
) *SessionUsecase {
	return &SessionUsecase{
		log:               log,
		RepositoryImports: ri,
	}
}

// LoadOnlineSessionMap map[nas_ip][]session.OnlineSession
func (u *SessionUsecase) LoadOnlineSessionMap(ts transaction.Session) (
	sessionMap map[session.NasIP][]session.OnlineSession, err error) {
	// получение списка онлайн сессий
	sessionList, err := u.Repository.Session.LoadOnlineSessionList(ts)
	switch {
	case err == nil:
		sessionMap = make(map[session.NasIP][]session.OnlineSession)

		// сортировка по nas_ip
		for _, sess := range sessionList {
			sessionMap[sess.NasIP] = append(sessionMap[sess.NasIP], sess)
		}

		return sessionMap, err
	case errors.Is(err, global.ErrNoData):
		return sessionMap, err
	default:
		u.log.Error("не удалось загрузить список онлайн сессий, ошибка", "error", err)
		err = global.ErrInternalError
		return sessionMap, err
	}
}
