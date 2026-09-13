package traffic_test

import (
	"net"
	"strings"
	"testing"

	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/entity/traffic"
	"aggregator/src/internal/transaction"
	"aggregator/src/rimport"
	"aggregator/src/tools/logger"
	"aggregator/src/uimport"

	"github.com/stretchr/testify/require"
	"github.com/yl2chen/cidranger"
	"go.uber.org/mock/gomock"
)

var (
	testLogger = logger.NewDiscard()
)

func TestAccumulateFlow(t *testing.T) {
	r := require.New(t)

	type fields struct {
		ri rimport.TestRepositoryImports
		ts *transaction.MockSession
	}
	type args struct {
		channelMap map[channel.ID]bool
		flow       string
	}

	const (
		ip1                 = "127.0.0.1"
		ip2                 = "127.0.0.2"
		disabledInternalRaw = "127.0.0.0/20"
	)

	sranger := cidranger.NewPCTrieRanger()
	_, network, err := net.ParseCIDR(disabledInternalRaw)
	r.NoError(err)
	sranger.Insert(cidranger.NewBasicRangerEntry(*network))

	tests := []struct {
		name string
		args args
		err  error
		data map[session.IP]map[channel.ID]traffic.Traffic
	}{
		{
			name: "подсчет internal сети",
			args: args{
				channelMap: map[channel.ID]bool{
					channel.Internal: true,
					channel.External: false,
				},
				flow: `132,127.0.0.1,127.0.0.2
456,127.0.0.2,127.0.0.1
234,127.0.0.1,127.0.0.2
345,127.0.0.2,127.0.0.1`,
			},
			err: nil,
			// выключенный external в мапку не попадает:
			// createNewEmptyTrafficMap заводит запись только по включенным каналам
			data: map[session.IP]map[channel.ID]traffic.Traffic{
				ip1: {
					channel.Internal: {
						Download: 366,
						Upload:   801,
					},
				},
				ip2: {
					channel.Internal: {
						Download: 801,
						Upload:   366,
					},
				},
			},
		},
		{
			name: "подсчет external сети",
			args: args{
				channelMap: map[channel.ID]bool{
					channel.External: true,
					channel.Internal: false,
				},
				flow: `534,127.0.0.1,34.249.117.10
347,34.249.117.10,127.0.0.1
7856,127.0.0.1,34.249.117.10
221,34.249.117.10,127.0.0.1`,
			},
			err: nil,
			data: map[session.IP]map[channel.ID]traffic.Traffic{
				ip1: {
					channel.External: {
						Download: 8390,
						Upload:   568,
					},
				},
			},
		},
		{
			name: "комплексный подсчет со всех сетей",
			args: args{
				channelMap: map[channel.ID]bool{
					channel.Internal: true,
					channel.External: true,
				},
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
			data: map[session.IP]map[channel.ID]traffic.Traffic{
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
				ip2: {
					channel.Internal: {
						Download: 801,
						Upload:   366,
					},
					channel.External: {
						Download: 0,
						Upload:   0,
					},
				},
			},
		},
		{
			// строки распарсились, но обе стороны во внешней сети — internal трафика нет.
			// это не сбой: Aggregate по ErrNoData имеет право убрать такой flow из tmp
			name: "flow только с external, internal трафика нет",
			args: args{
				channelMap: map[channel.ID]bool{
					channel.External: true,
					channel.Internal: false,
				},
				flow: `100,8.8.8.8,9.9.9.9
200,9.9.9.9,8.8.8.8`,
			},
			err:  global.ErrNoData,
			data: map[session.IP]map[channel.ID]traffic.Traffic{},
		},
		{
			// ни одной валидной строки (дрейф формата flow) — это сбой обработки,
			// Aggregate по ErrInternalError оставит flow в tmp, а не удалит
			name: "flow из нераспарсиваемых строк",
			args: args{
				channelMap: map[channel.ID]bool{
					channel.External: true,
					channel.Internal: true,
				},
				flow: `notanumber,127.0.0.1,127.0.0.2
also-broken,127.0.0.2,127.0.0.1`,
			},
			err:  global.ErrInternalError,
			data: map[session.IP]map[channel.ID]traffic.Traffic{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			f := fields{
				ri: rimport.NewTestRepositoryImports(ctrl),
				ts: transaction.NewMockSession(ctrl),
			}

			ui := uimport.NewUsecaseImports(testLogger, f.ri.RepositoryImports(), nil)

			acc := ui.Usecase.Traffic.NewFlowAccumulator(tt.args.channelMap)
			// строки подаются по одной, ровно как их отдает StreamFlow
			for _, line := range strings.Split(tt.args.flow, "\n") {
				r.NoError(acc.AccumulateLine(line))
			}

			data, errParse := acc.Result()
			r.Equal(tt.err, errParse)
			r.Equal(tt.data, data)
		})
	}
}

func TestCountTraffic(t *testing.T) {
	r := require.New(t)

	type fields struct {
		ri rimport.TestRepositoryImports
		ts *transaction.MockSession
	}
	type args struct {
		oldTraffic map[channel.ID]traffic.Traffic
		newTraffic traffic.Traffic
		channelMap map[channel.ID]bool
		channelID  channel.ID
	}

	tests := []struct {
		name    string
		prepare func(f *fields)
		args    args
		err     error
		data    map[channel.ID]traffic.Traffic
	}{
		{
			name:    "старый трафик существует",
			prepare: func(_ *fields) {},
			args: args{
				oldTraffic: map[channel.ID]traffic.Traffic{
					channel.Internal: {
						Download: 123,
						Upload:   20,
					},
					channel.External: {},
				},
				newTraffic: traffic.Traffic{
					Download: 7,
					Upload:   10,
				},
				channelMap: map[channel.ID]bool{
					channel.Internal: true,
					channel.External: false,
				},
				channelID: channel.Internal,
			},
			err: nil,
			data: map[channel.ID]traffic.Traffic{
				channel.Internal: {
					Download: 130,
					Upload:   30,
				},
				channel.External: {},
			},
		},
		{
			name: "старого трафика не существует",
			prepare: func(_ *fields) {
			},
			args: args{
				oldTraffic: map[channel.ID]traffic.Traffic{},
				newTraffic: traffic.Traffic{
					Download: 7,
					Upload:   10,
				},
				channelMap: map[channel.ID]bool{
					channel.Internal: true,
					channel.External: false,
				},
				channelID: channel.Internal,
			},
			err: nil,
			data: map[channel.ID]traffic.Traffic{ //nolint:exhaustive // только включённый канал уже отслеживается
				channel.Internal: {
					Download: 7,
					Upload:   10,
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
			}
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			ui := uimport.NewUsecaseImports(testLogger, f.ri.RepositoryImports(), nil)

			data := ui.Usecase.Traffic.CountTraffic(tt.args.oldTraffic, tt.args.newTraffic,
				tt.args.channelMap, tt.args.channelID)
			r.Equal(tt.data, data)
		})
	}
}

func TestSiftTraffic(t *testing.T) {
	r := require.New(t)

	type fields struct {
		ri rimport.TestRepositoryImports
		ts *transaction.MockSession
	}
	type args struct {
		channelMap  map[channel.ID]bool
		trafficMap  map[session.IP]map[channel.ID]traffic.Traffic
		sessionList []session.OnlineSession
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
		err     error
		data    []session.Chunk
	}{
		{
			name:    "трафика нет",
			prepare: func(_ *fields) {},
			args: args{
				channelMap: map[channel.ID]bool{
					channel.Internal: true,
					channel.External: false,
				},
				trafficMap: map[session.IP]map[channel.ID]traffic.Traffic{},
				sessionList: []session.OnlineSession{
					{
						SessID: sessID,
						IP:     ip1,
						NasIP:  nasIP,
					},
				},
			},
			err: nil,
			data: []session.Chunk{
				{
					SessID:    sessID,
					ChannelID: int(channel.Internal),
					Download:  0,
					Upload:    0,
				},
			},
		},
		{
			name:    "трафик есть",
			prepare: func(_ *fields) {},
			args: args{
				channelMap: map[channel.ID]bool{
					channel.Internal: true,
					channel.External: false,
				},
				trafficMap: map[session.IP]map[channel.ID]traffic.Traffic{
					ip1: {
						channel.Internal: {
							Download: 64,
							Upload:   2,
						},
					},
				},
				sessionList: []session.OnlineSession{
					{
						SessID: sessID,
						IP:     ip1,
						NasIP:  nasIP,
					},
				},
			},
			err: nil,
			data: []session.Chunk{
				{
					SessID:    sessID,
					ChannelID: int(channel.Internal),
					Download:  64,
					Upload:    2,
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
			}
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			ui := uimport.NewUsecaseImports(testLogger, f.ri.RepositoryImports(), nil)

			data, err := ui.Usecase.Traffic.SiftTraffic(tt.args.channelMap,
				tt.args.trafficMap, tt.args.sessionList)
			r.Equal(tt.err, err)
			r.Equal(tt.data, data)
		})
	}
}
