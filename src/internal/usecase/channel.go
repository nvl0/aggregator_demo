package usecase

import (
	"context"
	"errors"
	"log/slog"

	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/transaction"
	"aggregator/src/rimport"
)

type ChannelUsecase struct {
	log *slog.Logger
	//
	rimport.RepositoryImports
}

func NewChannelUsecase(
	log *slog.Logger,
	ri rimport.RepositoryImports,
) *ChannelUsecase {
	return &ChannelUsecase{
		log:               log,
		RepositoryImports: ri,
	}
}

// LoadChannelMap map[channel_id]enabled
func (u *ChannelUsecase) LoadChannelMap(ctx context.Context, ts transaction.Session) (
	channelMap map[channel.ID]bool, err error) {
	// получение списка каналов
	channelList, err := u.Repository.Channel.LoadChannelList(ctx, ts)
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
		u.log.ErrorContext(ctx, "не удалось загрузить список каналов, ошибка", "error", err)
		err = global.ErrInternalError
		return channelMap, err
	}
}
