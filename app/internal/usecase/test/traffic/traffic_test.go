package traffic_test

import (
	"aggregator/app/bimport"
	"aggregator/app/internal/entity/global"
	"aggregator/app/internal/entity/session"
	"aggregator/app/internal/entity/traffic"
	"aggregator/app/internal/transaction"
	"aggregator/app/rimport"
	"aggregator/app/tools/logger"
	"aggregator/app/uimport"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/yl2chen/cidranger"
)

var (
	testLogger = logger.NewNoFileLogger("test")
)

func TestParseFlow(t *testing.T) {
	r := assert.New(t)

	type fields struct {
		ri rimport.TestRepositoryImports
		bi *bimport.TestBridgeImports
		ts *transaction.MockSession
	}
	type args struct {
		flow string
	}

	const (
		disabledInternalRaw = "127.0.0.0/20"
	)

	tests := []struct {
		name    string
		prepare func(f *fields)
		args    args
		err     error
		data    map[string]map[global.ChannelID]traffic.Traffic
	}{
		{
			name: "подсчет internal сети",
			prepare: func(f *fields) {
				sranger := cidranger.NewPCTrieRanger()
				_, network, err := net.ParseCIDR(disabledInternalRaw)
				r.NoError(err)
				sranger.Insert(cidranger.NewBasicRangerEntry(*network))
			},
			args: args{
				flow: `132,127.0.0.1,127.0.0.2
456,127.0.0.2,127.0.0.1
234,127.0.0.1,127.0.0.2
345,127.0.0.2,127.0.0.1`,
			},
			err: nil,
			data: func() map[string]map[global.ChannelID]traffic.Traffic {

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

				return expData
			}(),
		},
		{
			name: "подсчет внешней сети",
			prepare: func(f *fields) {
				sranger := cidranger.NewPCTrieRanger()
				_, network, err := net.ParseCIDR(disabledInternalRaw)
				r.NoError(err)
				sranger.Insert(cidranger.NewBasicRangerEntry(*network))
			},
			args: args{
				flow: `534,127.0.0.1,34.249.117.10
347,34.249.117.10,127.0.0.1
7856,127.0.0.1,34.249.117.10
221,34.249.117.10,127.0.0.1`,
			},
			err: nil,
			data: func() map[string]map[global.ChannelID]traffic.Traffic {

				const ip1 = "127.0.0.1"

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
				}

				if global.EnabledChannelIDMap[global.Internet] {
					expData[ip1][global.Internet] = traffic.Traffic{
						Download: 8390,
						Upload:   568,
					}
				}

				return expData
			}(),
		},
		{
			name: "комплексный подсчет со всех сетей",
			prepare: func(f *fields) {
				sranger := cidranger.NewPCTrieRanger()
				_, network, err := net.ParseCIDR(disabledInternalRaw)
				r.NoError(err)
				sranger.Insert(cidranger.NewBasicRangerEntry(*network))
			},
			args: args{
				flow: `132,127.0.0.1,127.0.0.2
456,127.0.0.2,127.0.0.1
234,127.0.0.1,127.0.0.2
345,127.0.0.2,127.0.0.1
534,127.0.0.1,34.249.117.10
347,34.249.117.10,127.0.0.1
7856,127.0.0.1,34.249.117.10
221,34.249.117.10,127.0.0.1`,
			},
			err: nil,
			data: func() map[string]map[global.ChannelID]traffic.Traffic {

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
			}(),
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

			sm := transaction.NewMockSessionManager(ctrl)
			ui := uimport.NewUsecaseImports(testLogger, f.ri.RepositoryImports(), f.bi.BridgeImports(), sm)

			data, err := ui.Usecase.Traffic.ParseFlow(tt.args.flow)
			r.Equal(tt.err, err)
			r.Equal(tt.data, data)
		})
	}
}

func TestSiftTraffic(t *testing.T) {
	r := assert.New(t)

	type fields struct {
		ri rimport.TestRepositoryImports
		bi *bimport.TestBridgeImports
		ts *transaction.MockSession
	}
	type args struct {
		trafficMap  map[string]map[global.ChannelID]traffic.Traffic
		sessionList []session.Session
	}

	const (
		nasIP      = "127.0.0.0"
		ip1        = "127.0.0.1"
		sessID     = 1
		contractID = 2
		download   = 5233
	)

	trafficMap := map[string]map[global.ChannelID]traffic.Traffic{
		ip1: func() map[global.ChannelID]traffic.Traffic {
			channelMap := make(map[global.ChannelID]traffic.Traffic)

			channelMap[global.Internet] = traffic.NewTrafficDownload(download)

			return channelMap
		}(),
	}

	sessionList := []session.Session{
		{
			SessID: sessID,
			IP:     ip1,
			NasIP:  nasIP,
		},
	}

	tests := []struct {
		name    string
		prepare func(f *fields)
		args    args
		err     error
		data    []session.Chunk
	}{
		{
			name: "подсчет internal сети",
			prepare: func(f *fields) {
			},
			args: args{
				trafficMap:  trafficMap,
				sessionList: sessionList,
			},
			err: nil,
			data: []session.Chunk{
				{
					SessID:    sessID,
					ChannelID: int(global.Internet),
					Download:  download,
					Upload:    0,
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

			sm := transaction.NewMockSessionManager(ctrl)
			ui := uimport.NewUsecaseImports(testLogger, f.ri.RepositoryImports(), f.bi.BridgeImports(), sm)

			data, err := ui.Usecase.Traffic.SiftTraffic(tt.args.trafficMap, tt.args.sessionList)
			r.Equal(tt.err, err)
			r.Equal(tt.data, data)
		})
	}
}
