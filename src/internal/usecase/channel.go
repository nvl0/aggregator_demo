package usecase

import (
	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/transaction"
	"aggregator/src/rimport"

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
	channelMap map[channel.ChannelID]bool, err error) {

	// получение списка каналов
	channelList, err := u.Repository.Channel.LoadChannelList(ts)
	switch err {
	case nil:
		channelMap = make(map[channel.ChannelID]bool, len(channelList))

		for _, ch := range channelList {
			channelMap[ch.ID] = ch.Enabled
		}

		return
	case global.ErrNoData:
		return
	default:
		u.log.Errorln("не удалось загрузить список каналов, ошибка", err)
		err = global.ErrInternalError
		return
	}
}
