package aggregator_test

import (
	"aggregator/src/bimport"
	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/entity/traffic"
	"aggregator/src/internal/transaction"
	"aggregator/src/rimport"
	"aggregator/src/tools/logger"
	"aggregator/src/uimport"
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
)

var (
	testLogger = logger.NewNoFileLogger("test")
)

func TestStart(t *testing.T) {
	type fields struct {
		ri rimport.TestRepositoryImports
		bi *bimport.TestBridgeImports
		ts *transaction.MockSession
	}
	type args struct {
		ctx context.Context
	}

	const (
		nasIP  = "127.0.0.0"
		ip1    = "127.0.0.1"
		sessID = 1
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	tests := []struct {
		name    string
		prepare func(f *fields)
		args    args
	}{
		{
			name: "успешный результат",
			prepare: func(f *fields) {
				channelMap := map[channel.ChannelID]bool{
					channel.Internal: true,
				}
				sessionMap := map[session.NasIP][]session.OnlineSession{
					nasIP: {
						{
							SessID: sessID,
							NasIP:  nasIP,
							IP:     ip1,
						},
					},
				}
				dirList := []string{nasIP}

				f.ri.SessionManager.EXPECT().CreateSession().Return(f.ts).Times(2)
				f.ts.EXPECT().Start().Return(nil).Times(2)
				f.bi.TestBridge.Channel.EXPECT().LoadChannelMap(f.ts).Return(channelMap, nil)
				f.bi.TestBridge.Session.EXPECT().LoadOnlineSessionMap(f.ts).Return(sessionMap, nil)
				f.ts.EXPECT().Rollback().Return(nil).Times(2)

				f.ri.MockRepository.Flow.EXPECT().ReadFlowDirNames().Return(dirList, nil)

				for _, item := range dirList {
					f.bi.TestBridge.Aggregator.EXPECT().Aggregate(gomock.Any(), nasIP,
						sessionMap[session.NasIP(item)], channelMap)
				}
			},
			args: args{
				ctx: ctx,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				ri: rimport.NewTestRepositoryImports(ctrl),
				ts: transaction.NewMockSession(ctrl),
				bi: bimport.NewTestBridgeImports(ctrl),
			}
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			ui := uimport.NewUsecaseImports(testLogger, f.ri.RepositoryImports(), f.bi.BridgeImports())

			ui.Usecase.Aggregator.Start(tt.args.ctx)
		})
	}
}

func TestAggregate(t *testing.T) {
	type fields struct {
		ri rimport.TestRepositoryImports
		bi *bimport.TestBridgeImports
		ts *transaction.MockSession
	}
	type args struct {
		nasIP       string
		sessionList []session.OnlineSession
		channelMap  map[channel.ChannelID]bool
	}

	const (
		nasIP  = "127.0.0.0"
		ip1    = "127.0.0.1"
		sessID = 1
	)

	tests := []struct {
		name    string
		prepare func(f *fields)
		args    args
	}{
		{
			name: "успешный результат",
			prepare: func(f *fields) {
				flowStr :=
					`132,127.0.0.1,127.0.0.2
456,127.0.0.2,127.0.0.1
234,127.0.0.1,127.0.0.2
345,127.0.0.2,127.0.0.1
534,127.0.0.1,34.249.117.10
347,34.249.117.10,127.0.0.1
7856,127.0.0.1,34.249.117.10
221,34.249.117.10,127.0.0.1`

				channelMap := map[channel.ChannelID]bool{
					channel.Internal: true,
				}
				trafficMap := map[session.IP]map[channel.ChannelID]traffic.Traffic{
					ip1: {
						channel.Internal: {
							Download: 366,
							Upload:   801,
						},
						channel.External: {
							Download: 8390,
							Upload:   568,
						},
					},
				}
				sessionList := []session.OnlineSession{
					{
						SessID: sessID,
						NasIP:  nasIP,
						IP:     ip1,
					},
				}
				chunkList := []session.Chunk{
					{
						SessID:    sessID,
						ChannelID: int(channel.Internal),
						Download:  64,
						Upload:    2,
					},
				}

				gomock.InOrder(
					f.bi.TestBridge.Flow.EXPECT().PrepareFlow(nasIP).Return(flowStr, nil),
					f.bi.TestBridge.Traffic.EXPECT().ParseFlow(channelMap, flowStr).Return(trafficMap, nil),
					f.bi.TestBridge.Traffic.EXPECT().SiftTraffic(channelMap, trafficMap, sessionList).Return(chunkList, nil),
					f.ri.SessionManager.EXPECT().CreateSession().Return(f.ts),
					f.ts.EXPECT().Start().Return(nil),
					f.ri.MockRepository.Session.EXPECT().SaveChunkList(f.ts, chunkList).Return(nil),
					f.ts.EXPECT().Commit().Return(nil),
					f.ri.MockRepository.Flow.EXPECT().RemoveOld(nasIP).Return(nil),
					f.ts.EXPECT().Rollback().Return(nil),
				)
			},
			args: args{
				nasIP: nasIP,
				sessionList: []session.OnlineSession{
					{
						SessID: sessID,
						NasIP:  nasIP,
						IP:     ip1,
					},
				},
				channelMap: map[channel.ChannelID]bool{
					channel.Internal: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				ri: rimport.NewTestRepositoryImports(ctrl),
				ts: transaction.NewMockSession(ctrl),
				bi: bimport.NewTestBridgeImports(ctrl),
			}
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			ui := uimport.NewUsecaseImports(testLogger, f.ri.RepositoryImports(), f.bi.BridgeImports())

			var wg sync.WaitGroup
			wg.Add(1)

			ui.Usecase.Aggregator.Aggregate(&wg, tt.args.nasIP, tt.args.sessionList, tt.args.channelMap)
		})
	}
}
