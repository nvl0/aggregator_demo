package aggregator_test

import (
	"aggregator/app/bimport"
	"aggregator/app/internal/entity/global"
	"aggregator/app/internal/entity/session"
	"aggregator/app/internal/entity/traffic"
	"aggregator/app/internal/transaction"
	"aggregator/app/rimport"
	"aggregator/app/tools/logger"
	"aggregator/app/uimport"
	"testing"

	"github.com/golang/mock/gomock"
)

var (
	testLogger = logger.NewNoFileLogger("test")
)

func TestStart(t *testing.T) {
	type fields struct {
		ri rimport.TestRepositoryImports
		bi *bimport.TestBridgeImports
	}

	const (
		sessID     = 1
		contractID = 2
		ip         = "127.0.0.1"
		nasIP      = "127.0.0.0"
	)

	tests := []struct {
		name    string
		prepare func(f *fields)
	}{
		{
			name: "успешный результат",
			prepare: func(f *fields) {

				sessionMap := map[string][]session.Session{
					nasIP: {
						{
							SessID: sessID,
							IP:     ip,
							NasIP:  nasIP,
						},
					},
				}

				flowStr := `132,127.0.0.1,127.0.0.2
456,127.0.0.2,127.0.0.1
234,127.0.0.1,127.0.0.2
345,127.0.0.2,127.0.0.1
534,127.0.0.1,34.249.117.10
347,34.249.117.10,127.0.0.1
7856,127.0.0.1,34.249.117.10
221,34.249.117.10,127.0.0.1`

				trafficMap := func() map[string]map[global.ChannelID]traffic.Traffic {

					const (
						ip1 = "127.0.0.1"
						ip2 = "127.0.0.2"
					)

					expData := map[string]map[global.ChannelID]traffic.Traffic{
						ip1: func() map[global.ChannelID]traffic.Traffic {
							channelMap := make(map[global.ChannelID]traffic.Traffic)

							for _, channelID := range global.AllChannelIDList {
								if global.EnabledChannelIDMap[channelID] {
									channelMap[channelID] = traffic.NewEmptyTraffic()
								}
							}

							return channelMap
						}(),
						ip2: func() map[global.ChannelID]traffic.Traffic {
							channelMap := make(map[global.ChannelID]traffic.Traffic)

							for _, channelID := range global.AllChannelIDList {
								if global.EnabledChannelIDMap[channelID] {
									channelMap[channelID] = traffic.NewEmptyTraffic()
								}
							}

							return channelMap
						}(),
					}

					if global.EnabledChannelIDMap[global.Internal] {
						expData[ip1][global.Internal] = traffic.Traffic{
							Download: 366,
							Upload:   801,
						}

						expData[ip2][global.Internal] = traffic.Traffic{
							Download: 801,
							Upload:   366,
						}
					}

					if global.EnabledChannelIDMap[global.Internet] {
						expData[ip1][global.Internet] = traffic.Traffic{
							Download: 8390,
							Upload:   568,
						}

						expData[ip2][global.Internet] = traffic.Traffic{
							Download: 0,
							Upload:   0,
						}
					}

					return expData
				}()

				chunkList := func() []session.Chunk {
					data := make([]session.Chunk, 0)

					for _, channelID := range global.AllChannelIDList {
						if global.EnabledChannelIDMap[channelID] {
							switch channelID {
							case global.Internet:
								data = append(data, session.Chunk{
									SessID:    sessID,
									ChannelID: int(global.Internet),
									Download:  8390,
									Upload:    0,
								})
							case global.Internal:
								data = append(data, session.Chunk{
									SessID:    sessID,
									ChannelID: int(global.Internal),
									Download:  8390,
									Upload:    0,
								})
							}
						}
					}

					return data
				}()

				ts := f.ri.MockSessionWithCommit()

				f.ri.SessionManager.EXPECT().CreateSession().Return(ts)
				f.ri.MockRepository.Flow.EXPECT().ReadFlowDirNames().Return([]string{nasIP}, nil)
				f.bi.TestBridge.Session.EXPECT().LoadOnlineSessionListByNasIP(ts).Return(sessionMap, nil)

				f.bi.TestBridge.Flow.EXPECT().PrepareFlow(nasIP).Return(flowStr, nil).AnyTimes()
				f.bi.TestBridge.Traffic.EXPECT().ParseFlow(flowStr).Return(trafficMap, nil).AnyTimes()
				f.bi.TestBridge.Traffic.EXPECT().SiftTraffic(trafficMap, sessionMap[nasIP]).Return(chunkList, nil).AnyTimes()

				ts.EXPECT().CreateNewSession().Return(ts).AnyTimes()

				f.ri.MockRepository.Session.EXPECT().SaveChunkList(ts.CreateNewSession(), chunkList).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				ri: rimport.NewTestRepositoryImports(ctrl),
				bi: bimport.NewTestBridgeImports(ctrl),
			}
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			sm := transaction.NewMockSessionManager(ctrl)
			ui := uimport.NewUsecaseImports(testLogger, f.ri.RepositoryImports(), f.bi.BridgeImports(), sm)

			ui.Usecase.Aggregator.Start()
		})
	}
}
