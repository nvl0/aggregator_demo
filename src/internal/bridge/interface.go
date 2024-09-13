package bridge

import (
	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/entity/traffic"
	"aggregator/src/internal/transaction"
	"sync"
)

type Flow interface {
	PrepareFlow(dirName string) (flow string, err error)
}

type Session interface {
	LoadOnlineSessionMap(ts transaction.Session) (
		sessionMap map[session.NasIP][]session.OnlineSession, err error)
}

type Channel interface {
	LoadChannelMap(ts transaction.Session) (
		channelMap map[channel.ChannelID]bool, err error)
}

type Traffic interface {
	ParseFlow(channelMap map[channel.ChannelID]bool, flow string) (
		trafficMap map[session.IP]map[channel.ChannelID]traffic.Traffic, err error)
	CountTraffic(oldTraffic map[channel.ChannelID]traffic.Traffic,
		newTraffic traffic.Traffic, channelMap map[channel.ChannelID]bool,
		channelID channel.ChannelID) map[channel.ChannelID]traffic.Traffic
	SiftTraffic(channelMap map[channel.ChannelID]bool, trafficMap map[session.IP]map[channel.ChannelID]traffic.Traffic,
		sessionList []session.OnlineSession) (chunkList []session.Chunk, err error)
}

type Aggregator interface {
	Aggregate(wg *sync.WaitGroup, nasIP string, sessionList []session.OnlineSession,
		channelMap map[channel.ChannelID]bool)
}
