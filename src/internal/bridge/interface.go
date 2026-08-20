package bridge

import (
	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/entity/traffic"
	"aggregator/src/internal/transaction"
)

type Flow interface {
	PrepareFlow(dirName string, skipFileNames map[string]bool) (
		flow string, fileNameList []string, err error)
}

type Session interface {
	LoadOnlineSessionMap(ts transaction.Session) (
		sessionMap map[session.NasIP][]session.OnlineSession, err error)
}

type Channel interface {
	LoadChannelMap(ts transaction.Session) (
		channelMap map[channel.ID]bool, err error)
}

type Traffic interface {
	ParseFlow(channelMap map[channel.ID]bool, flow string) (
		trafficMap map[session.IP]map[channel.ID]traffic.Traffic, err error)
	CountTraffic(oldTraffic map[channel.ID]traffic.Traffic,
		newTraffic traffic.Traffic, channelMap map[channel.ID]bool,
		channelID channel.ID) map[channel.ID]traffic.Traffic
	SiftTraffic(channelMap map[channel.ID]bool, trafficMap map[session.IP]map[channel.ID]traffic.Traffic,
		sessionList []session.OnlineSession) (chunkList []session.Chunk, err error)
}

type Aggregator interface {
	Aggregate(nasIP string, sessionList []session.OnlineSession,
		channelMap map[channel.ID]bool)
}
