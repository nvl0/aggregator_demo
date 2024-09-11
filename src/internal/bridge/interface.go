package bridge

import (
	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/entity/traffic"
	"aggregator/src/internal/transaction"
)

type Flow interface {
	PrepareFlow(dirName string) (flow string, err error)
}

type Session interface {
	LoadOnlineSessionMap(ts transaction.Session) (
		sessionMap map[string][]session.OnlineSession, err error)
}

type Channel interface {
	LoadChannelMap(ts transaction.Session) (
		channelMap map[channel.ChannelID]bool, err error)
}

type Traffic interface {
	ParseFlow(channelMap map[channel.ChannelID]bool, flow string) (
		trafficMap map[string]map[channel.ChannelID]traffic.Traffic, err error)
	CountTraffic(oldTraffic map[channel.ChannelID]traffic.Traffic,
		newTraffic traffic.Traffic, channelMap map[channel.ChannelID]bool,
		channelID channel.ChannelID) map[channel.ChannelID]traffic.Traffic
	SiftTraffic(channelMap map[channel.ChannelID]bool, trafficMap map[string]map[channel.ChannelID]traffic.Traffic,
		sessionList []session.OnlineSession) (chunkList []session.Chunk, err error)
}
