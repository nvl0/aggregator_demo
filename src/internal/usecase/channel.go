package usecase

import (
	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/transaction"
	"aggregator/src/rimport"
	"errors"

	"github.com/sirupsen/logrus"
)

type ChannelUsecase struct {
	log *logrus.Logger
	//
	rimport.RepositoryImports
}

func NewChannelUsecase(
	log *logrus.Logger,
	ri rimport.RepositoryImports,
) *ChannelUsecase {
	return &ChannelUsecase{
		log:               log,
		RepositoryImports: ri,
	}
}

// LoadChannelMap map[channel_id]enabled
func (u *ChannelUsecase) LoadChannelMap(ts transaction.Session) (
	channelMap map[channel.ID]bool, err error) {
	// получение списка каналов
	channelList, err := u.Repository.Channel.LoadChannelList(ts)
	switch {
	case err == nil:
		channelMap = make(map[channel.ID]bool, len(channelList))

		for _, ch := range channelList {
			channelMap[ch.ID] = ch.Enabled
		}

		return channelMap, err
	case errors.Is(err, global.ErrNoData):
		return channelMap, err
	default:
		u.log.Errorln("не удалось загрузить список каналов, ошибка", err)
		err = global.ErrInternalError
		return channelMap, err
	}
}
