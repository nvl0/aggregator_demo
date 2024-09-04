package postgresql

import (
	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/repository"
	"aggregator/src/internal/transaction"
	"aggregator/src/tools/gensql"
)

type channelRepository struct {
}

func NewChannelRepository() repository.Channel {
	return &channelRepository{}
}

// LoadOnlineSessionList загрузить список каналов
func (r *channelRepository) LoadChannelList(ts transaction.Session) ([]channel.Channel, error) {
	sqlQuery := `
		select c.channel_id, c.enabled, c.descr
		from channel c
		order by c.channel_id`

	return gensql.Select[channel.Channel](SqlxTx(ts), sqlQuery)
}
