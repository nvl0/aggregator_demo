package usecase

import (
	"context"

	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/entity/traffic"
	"aggregator/src/internal/transaction"
)

// FlowPreparer подготовка flow, реализация — *FlowUsecase
type FlowPreparer interface {
	PrepareFlow(dirName string, skipFileNames map[string]bool) (
		flow string, fileNameList []string, err error)
}

// SessionLoader загрузка онлайн сессий, реализация — *SessionUsecase
type SessionLoader interface {
	LoadOnlineSessionMap(ctx context.Context, ts transaction.Session) (
		sessionMap map[session.NasIP][]session.OnlineSession, err error)
}

// ChannelLoader загрузка каналов, реализация — *ChannelUsecase
type ChannelLoader interface {
	LoadChannelMap(ctx context.Context, ts transaction.Session) (
		channelMap map[channel.ID]bool, err error)
}

// TrafficProcessor разбор flow и просеивание трафика, реализация — *TrafficUsecase
type TrafficProcessor interface {
	ParseFlow(channelMap map[channel.ID]bool, flow string) (
		trafficMap map[session.IP]map[channel.ID]traffic.Traffic, err error)
	SiftTraffic(channelMap map[channel.ID]bool,
		trafficMap map[session.IP]map[channel.ID]traffic.Traffic,
		sessionList []session.OnlineSession) (chunkList []session.Chunk, err error)
}
