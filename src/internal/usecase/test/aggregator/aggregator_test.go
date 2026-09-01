package aggregator_test

import (
	"context"
	"errors"
	"testing"

	"aggregator/src/bimport"
	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/entity/traffic"
	"aggregator/src/internal/transaction"
	"aggregator/src/rimport"
	"aggregator/src/tools/logger"
	"aggregator/src/uimport"

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

	canceledCtx, cancelNow := context.WithCancel(context.Background())
	cancelNow()

	tests := []struct {
		name    string
		prepare func(f *fields)
		args    args
	}{
		{
			name: "успешный результат",
			prepare: func(f *fields) {
				channelMap := map[channel.ID]bool{
					channel.Internal: true,
					channel.External: false,
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
					f.bi.TestBridge.Aggregator.EXPECT().Aggregate(nasIP,
						sessionMap[item], channelMap)
				}
			},
			args: args{
				ctx: context.Background(),
			},
		},
		{
			name: "контекст отменен, рассылка не выполняется",
			prepare: func(f *fields) {
				channelMap := map[channel.ID]bool{
					channel.Internal: true,
					channel.External: false,
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

				// Aggregate не ожидается: пул возвращает false на отмененном контексте
			},
			args: args{
				ctx: canceledCtx,
			},
		},
		{
			name: "паника воркера не роняет рассылку",
			prepare: func(f *fields) {
				channelMap := map[channel.ID]bool{
					channel.Internal: true,
					channel.External: false,
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

				// если пул не перехватит панику, упадет весь тестовый процесс
				f.bi.TestBridge.Aggregator.EXPECT().
					Aggregate(nasIP, sessionMap[nasIP], channelMap).
					Do(func(_ string, _ []session.OnlineSession, _ map[channel.ID]bool) {
						panic("паника тестового воркера")
					})
			},
			args: args{
				ctx: context.Background(),
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
		channelMap  map[channel.ID]bool
	}

	const (
		nasIP   = "127.0.0.0"
		ip1     = "127.0.0.1"
		sessID  = 1
		newFile = "ft-01.01.2026-00:05:00"
		oldFile = "ft-01.01.2026-00:00:00"
	)

	flowStr :=
		`132,127.0.0.1,127.0.0.2
456,127.0.0.2,127.0.0.1
234,127.0.0.1,127.0.0.2
345,127.0.0.2,127.0.0.1
534,127.0.0.1,34.249.117.10
347,34.249.117.10,127.0.0.1
7856,127.0.0.1,34.249.117.10
221,34.249.117.10,127.0.0.1`

	channelMap := map[channel.ID]bool{
		channel.Internal: true,
		channel.External: false,
	}
	trafficMap := map[session.IP]map[channel.ID]traffic.Traffic{
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

	tests := []struct {
		name    string
		prepare func(f *fields)
		args    args
	}{
		{
			name: "обычный цикл без чекпоинта",
			prepare: func(f *fields) {
				committedFileNames := map[string]bool{}
				fileNameList := []string{newFile}

				// три транзакции: загрузка чекпоинта, сохранение чанков, удаление чекпоинта
				f.ri.SessionManager.EXPECT().CreateSession().Return(f.ts).Times(3)
				f.ts.EXPECT().Start().Return(nil).Times(3)
				f.ts.EXPECT().Rollback().Return(nil).Times(3)

				gomock.InOrder(
					f.ri.MockRepository.FlowBatch.EXPECT().
						LoadCommittedFileNames(f.ts, nasIP).Return(committedFileNames, nil),
					f.bi.TestBridge.Flow.EXPECT().
						PrepareFlow(nasIP, committedFileNames).Return(flowStr, fileNameList, nil),
					f.bi.TestBridge.Traffic.EXPECT().
						ParseFlow(channelMap, flowStr).Return(trafficMap, nil),
					f.bi.TestBridge.Traffic.EXPECT().
						SiftTraffic(channelMap, trafficMap, sessionList).Return(chunkList, nil),
					f.ri.MockRepository.Session.EXPECT().SaveChunkList(f.ts, chunkList).Return(nil),
					f.ri.MockRepository.FlowBatch.EXPECT().
						SaveFileNames(f.ts, nasIP, fileNameList).Return(nil),
					f.ts.EXPECT().Commit().Return(nil),
					f.ri.MockRepository.Flow.EXPECT().RemoveOld(nasIP).Return(nil),
					f.ri.MockRepository.FlowBatch.EXPECT().RemoveByNasIP(f.ts, nasIP).Return(nil),
					f.ts.EXPECT().Commit().Return(nil),
				)
			},
			args: args{
				nasIP:       nasIP,
				sessionList: sessionList,
				channelMap:  channelMap,
			},
		},
		{
			name: "чистый replay, чанки повторно не пишутся",
			prepare: func(f *fields) {
				committedFileNames := map[string]bool{oldFile: true}
				fileNameList := []string{oldFile}

				// две транзакции: загрузка чекпоинта и его удаление,
				// транзакция сохранения чанков не открывается
				f.ri.SessionManager.EXPECT().CreateSession().Return(f.ts).Times(2)
				f.ts.EXPECT().Start().Return(nil).Times(2)
				f.ts.EXPECT().Rollback().Return(nil).Times(2)

				gomock.InOrder(
					f.ri.MockRepository.FlowBatch.EXPECT().
						LoadCommittedFileNames(f.ts, nasIP).Return(committedFileNames, nil),
					f.bi.TestBridge.Flow.EXPECT().
						PrepareFlow(nasIP, committedFileNames).Return("", fileNameList, nil),
					f.ri.MockRepository.Flow.EXPECT().RemoveOld(nasIP).Return(nil),
					f.ri.MockRepository.FlowBatch.EXPECT().RemoveByNasIP(f.ts, nasIP).Return(nil),
					f.ts.EXPECT().Commit().Return(nil),
				)
			},
			args: args{
				nasIP:       nasIP,
				sessionList: sessionList,
				channelMap:  channelMap,
			},
		},
		{
			name: "смешанный набор, в чекпоинт пишется полный список",
			prepare: func(f *fields) {
				committedFileNames := map[string]bool{oldFile: true}
				fileNameList := []string{oldFile, newFile}

				f.ri.SessionManager.EXPECT().CreateSession().Return(f.ts).Times(3)
				f.ts.EXPECT().Start().Return(nil).Times(3)
				f.ts.EXPECT().Rollback().Return(nil).Times(3)

				gomock.InOrder(
					f.ri.MockRepository.FlowBatch.EXPECT().
						LoadCommittedFileNames(f.ts, nasIP).Return(committedFileNames, nil),
					// flowStr содержит только новый файл, старый пропущен в ReadFlow
					f.bi.TestBridge.Flow.EXPECT().
						PrepareFlow(nasIP, committedFileNames).Return(flowStr, fileNameList, nil),
					f.bi.TestBridge.Traffic.EXPECT().
						ParseFlow(channelMap, flowStr).Return(trafficMap, nil),
					f.bi.TestBridge.Traffic.EXPECT().
						SiftTraffic(channelMap, trafficMap, sessionList).Return(chunkList, nil),
					f.ri.MockRepository.Session.EXPECT().SaveChunkList(f.ts, chunkList).Return(nil),
					// в чекпоинт уходит и уже известный oldFile, и новый newFile
					f.ri.MockRepository.FlowBatch.EXPECT().
						SaveFileNames(f.ts, nasIP, fileNameList).Return(nil),
					f.ts.EXPECT().Commit().Return(nil),
					f.ri.MockRepository.Flow.EXPECT().RemoveOld(nasIP).Return(nil),
					f.ri.MockRepository.FlowBatch.EXPECT().RemoveByNasIP(f.ts, nasIP).Return(nil),
					f.ts.EXPECT().Commit().Return(nil),
				)
			},
			args: args{
				nasIP:       nasIP,
				sessionList: sessionList,
				channelMap:  channelMap,
			},
		},
		{
			name: "ошибка RemoveOld, чекпоинт не удаляется",
			prepare: func(f *fields) {
				committedFileNames := map[string]bool{}
				fileNameList := []string{newFile}

				// две транзакции: загрузка чекпоинта и сохранение чанков,
				// транзакция удаления чекпоинта не открывается
				f.ri.SessionManager.EXPECT().CreateSession().Return(f.ts).Times(2)
				f.ts.EXPECT().Start().Return(nil).Times(2)
				f.ts.EXPECT().Rollback().Return(nil).Times(2)

				gomock.InOrder(
					f.ri.MockRepository.FlowBatch.EXPECT().
						LoadCommittedFileNames(f.ts, nasIP).Return(committedFileNames, nil),
					f.bi.TestBridge.Flow.EXPECT().
						PrepareFlow(nasIP, committedFileNames).Return(flowStr, fileNameList, nil),
					f.bi.TestBridge.Traffic.EXPECT().
						ParseFlow(channelMap, flowStr).Return(trafficMap, nil),
					f.bi.TestBridge.Traffic.EXPECT().
						SiftTraffic(channelMap, trafficMap, sessionList).Return(chunkList, nil),
					f.ri.MockRepository.Session.EXPECT().SaveChunkList(f.ts, chunkList).Return(nil),
					f.ri.MockRepository.FlowBatch.EXPECT().
						SaveFileNames(f.ts, nasIP, fileNameList).Return(nil),
					f.ts.EXPECT().Commit().Return(nil),
					f.ri.MockRepository.Flow.EXPECT().
						RemoveOld(nasIP).Return(errors.New("диск недоступен")),
				)

				// RemoveByNasIP не ожидается: чекпоинт обязан пережить неудачную очистку
			},
			args: args{
				nasIP:       nasIP,
				sessionList: sessionList,
				channelMap:  channelMap,
			},
		},
		{
			name: "ParseFlow без учитываемого трафика, tmp очищается без чекпоинта",
			prepare: func(f *fields) {
				committedFileNames := map[string]bool{}
				fileNameList := []string{newFile}

				// две транзакции: загрузка чекпоинта и его удаление при очистке tmp,
				// транзакция сохранения чанков не открывается
				f.ri.SessionManager.EXPECT().CreateSession().Return(f.ts).Times(2)
				f.ts.EXPECT().Start().Return(nil).Times(2)
				f.ts.EXPECT().Rollback().Return(nil).Times(2)

				gomock.InOrder(
					f.ri.MockRepository.FlowBatch.EXPECT().
						LoadCommittedFileNames(f.ts, nasIP).Return(committedFileNames, nil),
					f.bi.TestBridge.Flow.EXPECT().
						PrepareFlow(nasIP, committedFileNames).Return(flowStr, fileNameList, nil),
					f.bi.TestBridge.Traffic.EXPECT().
						ParseFlow(channelMap, flowStr).Return(nil, global.ErrNoData),
					// файлы уже в tmp: их обязательно нужно убрать, иначе они копятся
					f.ri.MockRepository.Flow.EXPECT().RemoveOld(nasIP).Return(nil),
					f.ri.MockRepository.FlowBatch.EXPECT().RemoveByNasIP(f.ts, nasIP).Return(nil),
					f.ts.EXPECT().Commit().Return(nil),
				)

				// SaveChunkList / SaveFileNames не ожидаются: сохранять нечего
			},
			args: args{
				nasIP:       nasIP,
				sessionList: sessionList,
				channelMap:  channelMap,
			},
		},
		{
			name: "SiftTraffic без данных, tmp очищается без чекпоинта",
			prepare: func(f *fields) {
				committedFileNames := map[string]bool{}
				fileNameList := []string{newFile}

				f.ri.SessionManager.EXPECT().CreateSession().Return(f.ts).Times(2)
				f.ts.EXPECT().Start().Return(nil).Times(2)
				f.ts.EXPECT().Rollback().Return(nil).Times(2)

				gomock.InOrder(
					f.ri.MockRepository.FlowBatch.EXPECT().
						LoadCommittedFileNames(f.ts, nasIP).Return(committedFileNames, nil),
					f.bi.TestBridge.Flow.EXPECT().
						PrepareFlow(nasIP, committedFileNames).Return(flowStr, fileNameList, nil),
					f.bi.TestBridge.Traffic.EXPECT().
						ParseFlow(channelMap, flowStr).Return(trafficMap, nil),
					f.bi.TestBridge.Traffic.EXPECT().
						SiftTraffic(channelMap, trafficMap, sessionList).Return(nil, global.ErrNoData),
					f.ri.MockRepository.Flow.EXPECT().RemoveOld(nasIP).Return(nil),
					f.ri.MockRepository.FlowBatch.EXPECT().RemoveByNasIP(f.ts, nasIP).Return(nil),
					f.ts.EXPECT().Commit().Return(nil),
				)
			},
			args: args{
				nasIP:       nasIP,
				sessionList: sessionList,
				channelMap:  channelMap,
			},
		},
		{
			name: "ParseFlow не распознал flow, tmp не трогаем",
			prepare: func(f *fields) {
				committedFileNames := map[string]bool{}
				fileNameList := []string{newFile}

				// одна транзакция: только загрузка чекпоинта.
				// очистка tmp не вызывается — flow должен остаться на диске
				f.ri.SessionManager.EXPECT().CreateSession().Return(f.ts).Times(1)
				f.ts.EXPECT().Start().Return(nil).Times(1)
				f.ts.EXPECT().Rollback().Return(nil).Times(1)

				gomock.InOrder(
					f.ri.MockRepository.FlowBatch.EXPECT().
						LoadCommittedFileNames(f.ts, nasIP).Return(committedFileNames, nil),
					f.bi.TestBridge.Flow.EXPECT().
						PrepareFlow(nasIP, committedFileNames).Return(flowStr, fileNameList, nil),
					f.bi.TestBridge.Traffic.EXPECT().
						ParseFlow(channelMap, flowStr).Return(nil, global.ErrInternalError),
				)

				// RemoveOld / RemoveByNasIP не ожидаются
			},
			args: args{
				nasIP:       nasIP,
				sessionList: sessionList,
				channelMap:  channelMap,
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

			ui.Usecase.Aggregator.Aggregate(tt.args.nasIP, tt.args.sessionList, tt.args.channelMap)
		})
	}
}
