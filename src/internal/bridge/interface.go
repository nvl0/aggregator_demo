package bridge

import (
	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/entity/traffic"
	"aggregator/src/internal/transaction"
)

type Flow interface {
	PrepareFlow(dirName string) (flow string, err error)
}

type Session interface {
	LoadOnlineSessionListByNasIP(ts transaction.Session) (
		sessionMap map[string][]session.Session, err error)
}

type Traffic interface {
	ParseFlow(flow string) (trafficMap map[string]map[global.ChannelID]traffic.Traffic, err error)
	SiftTraffic(trafficMap map[string]map[global.ChannelID]traffic.Traffic,
		sessionList []session.Session) (chunkList []session.Chunk, err error)
}
