package usecase

import (
	"context"

	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/entity/traffic"
	"aggregator/src/internal/transaction"
)

// FlowStreamer подготовка и потоковое чтение flow, реализация — *FlowUsecase
type FlowStreamer interface {
	StreamFlow(dirName string, skipFileNames map[string]bool, onLine func(line string) error) (
		fileNameList []string, flowSize int, err error)
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
	NewFlowAccumulator(channelMap map[channel.ID]bool) *FlowAccumulator
	SiftTraffic(channelMap map[channel.ID]bool,
		trafficMap map[session.IP]map[channel.ID]traffic.Traffic,
		sessionList []session.OnlineSession) (chunkList []session.Chunk, err error)
}
