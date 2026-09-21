package aggregator_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/entity/traffic"
	"aggregator/src/internal/transaction"
	"aggregator/src/internal/usecase"
	"aggregator/src/rimport"
	"aggregator/src/tools/logger"
	"aggregator/src/tools/metrics"
	"aggregator/src/tools/tracing"

	"github.com/yl2chen/cidranger"
	"go.uber.org/mock/gomock"
)

var (
	testLogger = logger.NewDiscard()
)

// newTestTrafficUsecase настоящий TrafficUsecase поверх подсети internal 127.0.0.0/20.
// из него берется живой FlowAccumulator: разбор строк внутри callback мокировать нечем
func newTestTrafficUsecase(t *testing.T, ri rimport.RepositoryImports) *usecase.TrafficUsecase {
	t.Helper()

	ranger := cidranger.NewPCTrieRanger()

	_, network, err := net.ParseCIDR("127.0.0.0/20")
	if err != nil {
		t.Fatalf("не удалось разобрать подсеть internal, ошибка %v", err)
	}

	if err = ranger.Insert(cidranger.NewBasicRangerEntry(*network)); err != nil {
		t.Fatalf("не удалось наполнить подсеть internal, ошибка %v", err)
	}

	return usecase.NewTrafficUsecase(testLogger, ri, ranger)
}

// streamLines DoAndReturn для мока StreamFlow: прогоняет flow построчно
// через callback агрегатора и возвращает список файлов и размер flow
func streamLines(flowStr string, fileNameList []string) func(
	string, map[string]bool, func(line string) error) ([]string, int, error) {
	return func(_ string, _ map[string]bool, onLine func(line string) error) (
		[]string, int, error) {
		for _, line := range strings.Split(flowStr, "\n") {
			if err := onLine(line); err != nil {
				return fileNameList, 0, err
			}
		}

		return fileNameList, len(flowStr), nil
	}
}

func TestStart(t *testing.T) {
	type fields struct {
		ri      rimport.TestRepositoryImports
		ts      *transaction.MockSession
		flow    *usecase.MockFlowStreamer
		session *usecase.MockSessionLoader
		channel *usecase.MockChannelLoader
		traffic *usecase.MockTrafficProcessor
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

				// три транзакции: мапка каналов, мапка сессий
				// и чекпоинт внутри дошедшего до воркера Aggregate
				f.ri.SessionManager.EXPECT().CreateSession().Return(f.ts).Times(3)
				f.ts.EXPECT().Start(gomock.Any()).Return(nil).Times(3)
				f.channel.EXPECT().LoadChannelMap(gomock.Any(), f.ts).Return(channelMap, nil)
				f.session.EXPECT().LoadOnlineSessionMap(gomock.Any(), f.ts).Return(sessionMap, nil)
				f.ts.EXPECT().Rollback().Return(nil).Times(3)

				f.ri.MockRepository.Flow.EXPECT().ReadFlowDirNames().Return(dirList, nil)

				// Aggregate больше не мок: это прямой самовызов.
				// рассылка дошла до воркера, если начался первый шаг Aggregate —
				// загрузка чекпоинта. Ошибка обрывает обработку nas_ip сразу после неё
				f.ri.MockRepository.FlowBatch.EXPECT().
					LoadCommittedFileNames(gomock.Any(), f.ts, nasIP).
					Return(nil, errors.New("проверка рассылки"))
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
				f.ts.EXPECT().Start(gomock.Any()).Return(nil).Times(2)
				f.channel.EXPECT().LoadChannelMap(gomock.Any(), f.ts).Return(channelMap, nil)
				f.session.EXPECT().LoadOnlineSessionMap(gomock.Any(), f.ts).Return(sessionMap, nil)
				f.ts.EXPECT().Rollback().Return(nil).Times(2)

				f.ri.MockRepository.Flow.EXPECT().ReadFlowDirNames().Return(dirList, nil)

				// пул возвращает false на отмененном контексте: воркер не стартует,
				// значит и до загрузки чекпоинта дело не доходит
				f.ri.MockRepository.FlowBatch.EXPECT().
					LoadCommittedFileNames(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
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

				f.ri.SessionManager.EXPECT().CreateSession().Return(f.ts).Times(3)
				f.ts.EXPECT().Start(gomock.Any()).Return(nil).Times(3)
				f.channel.EXPECT().LoadChannelMap(gomock.Any(), f.ts).Return(channelMap, nil)
				f.session.EXPECT().LoadOnlineSessionMap(gomock.Any(), f.ts).Return(sessionMap, nil)
				// rollback чекпоинтной транзакции докатывается на раскрутке паники
				f.ts.EXPECT().Rollback().Return(nil).Times(3)

				f.ri.MockRepository.Flow.EXPECT().ReadFlowDirNames().Return(dirList, nil)

				// если пул не перехватит панику, упадет весь тестовый процесс
				f.ri.MockRepository.FlowBatch.EXPECT().
					LoadCommittedFileNames(gomock.Any(), f.ts, nasIP).
					Do(func(_ context.Context, _ transaction.Session, _ string) {
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
				ri:      rimport.NewTestRepositoryImports(ctrl),
				ts:      transaction.NewMockSession(ctrl),
				flow:    usecase.NewMockFlowStreamer(ctrl),
				session: usecase.NewMockSessionLoader(ctrl),
				channel: usecase.NewMockChannelLoader(ctrl),
				traffic: usecase.NewMockTrafficProcessor(ctrl),
			}
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			u := usecase.NewAggregatorUsecase(testLogger, f.ri.RepositoryImports(),
				usecase.AggregatorDeps{
					Flow:    f.flow,
					Session: f.session,
					Channel: f.channel,
					Traffic: f.traffic,
				}, metrics.Nop(), tracing.Nop())

			u.Start(tt.args.ctx)
		})
	}
}

func TestAggregate(t *testing.T) {
	type fields struct {
		ri      rimport.TestRepositoryImports
		ts      *transaction.MockSession
		flow    *usecase.MockFlowStreamer
		session *usecase.MockSessionLoader
		channel *usecase.MockChannelLoader
		traffic *usecase.MockTrafficProcessor
		// realTraffic источник живого FlowAccumulator для мока TrafficProcessor
		realTraffic *usecase.TrafficUsecase
	}
	type args struct {
		ctx         context.Context
		nasIP       string
		sessionList []session.OnlineSession
		channelMap  map[channel.ID]bool
	}

	const (
		nasIP   = "127.0.0.0"
		ip1     = "127.0.0.1"
		ip2     = "127.0.0.2"
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
	// то, что реально накопит FlowAccumulator из flowStr при channelMap кейсов:
	// external выключен, поэтому в мапке только internal
	trafficMap := map[session.IP]map[channel.ID]traffic.Traffic{
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
	}
	// обе стороны во внешней сети: строки валидны, но internal трафика нет
	externalOnlyFlow := `100,8.8.8.8,9.9.9.9
200,9.9.9.9,8.8.8.8`

	// ни одной валидной строки: дрейф формата flow
	brokenFlow := `notanumber,127.0.0.1,127.0.0.2
also-broken,127.0.0.2,127.0.0.1`

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
				f.ts.EXPECT().Start(gomock.Any()).Return(nil).Times(3)
				f.ts.EXPECT().Rollback().Return(nil).Times(3)

				gomock.InOrder(
					f.ri.MockRepository.FlowBatch.EXPECT().
						LoadCommittedFileNames(gomock.Any(), f.ts, nasIP).Return(committedFileNames, nil),
					f.traffic.EXPECT().NewFlowAccumulator(channelMap).
						Return(f.realTraffic.NewFlowAccumulator(channelMap)),
					f.flow.EXPECT().StreamFlow(nasIP, committedFileNames, gomock.Any()).
						DoAndReturn(streamLines(flowStr, fileNameList)),
					f.traffic.EXPECT().
						SiftTraffic(channelMap, trafficMap, sessionList).Return(chunkList, nil),
					f.ri.MockRepository.Session.EXPECT().SaveChunkList(gomock.Any(), f.ts, chunkList).Return(nil),
					f.ri.MockRepository.FlowBatch.EXPECT().
						SaveFileNames(gomock.Any(), f.ts, nasIP, fileNameList).Return(nil),
					f.ts.EXPECT().Commit().Return(nil),
					f.ri.MockRepository.Flow.EXPECT().RemoveOld(nasIP).Return(nil),
					f.ri.MockRepository.FlowBatch.EXPECT().RemoveByNasIP(gomock.Any(), f.ts, nasIP).Return(nil),
					f.ts.EXPECT().Commit().Return(nil),
				)
			},
			args: args{
				ctx:         context.Background(),
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
				f.ts.EXPECT().Start(gomock.Any()).Return(nil).Times(2)
				f.ts.EXPECT().Rollback().Return(nil).Times(2)

				gomock.InOrder(
					f.ri.MockRepository.FlowBatch.EXPECT().
						LoadCommittedFileNames(gomock.Any(), f.ts, nasIP).Return(committedFileNames, nil),
					f.traffic.EXPECT().NewFlowAccumulator(channelMap).
						Return(f.realTraffic.NewFlowAccumulator(channelMap)),
					f.flow.EXPECT().StreamFlow(nasIP, committedFileNames, gomock.Any()).
						Return(fileNameList, 0, nil),
					f.ri.MockRepository.Flow.EXPECT().RemoveOld(nasIP).Return(nil),
					f.ri.MockRepository.FlowBatch.EXPECT().RemoveByNasIP(gomock.Any(), f.ts, nasIP).Return(nil),
					f.ts.EXPECT().Commit().Return(nil),
				)
			},
			args: args{
				ctx:         context.Background(),
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
				f.ts.EXPECT().Start(gomock.Any()).Return(nil).Times(3)
				f.ts.EXPECT().Rollback().Return(nil).Times(3)

				gomock.InOrder(
					f.ri.MockRepository.FlowBatch.EXPECT().
						LoadCommittedFileNames(gomock.Any(), f.ts, nasIP).Return(committedFileNames, nil),
					f.traffic.EXPECT().NewFlowAccumulator(channelMap).
						Return(f.realTraffic.NewFlowAccumulator(channelMap)),
					// flowStr содержит только новый файл, старый пропущен в StreamFlow
					f.flow.EXPECT().StreamFlow(nasIP, committedFileNames, gomock.Any()).
						DoAndReturn(streamLines(flowStr, fileNameList)),
					f.traffic.EXPECT().
						SiftTraffic(channelMap, trafficMap, sessionList).Return(chunkList, nil),
					f.ri.MockRepository.Session.EXPECT().SaveChunkList(gomock.Any(), f.ts, chunkList).Return(nil),
					// в чекпоинт уходит и уже известный oldFile, и новый newFile
					f.ri.MockRepository.FlowBatch.EXPECT().
						SaveFileNames(gomock.Any(), f.ts, nasIP, fileNameList).Return(nil),
					f.ts.EXPECT().Commit().Return(nil),
					f.ri.MockRepository.Flow.EXPECT().RemoveOld(nasIP).Return(nil),
					f.ri.MockRepository.FlowBatch.EXPECT().RemoveByNasIP(gomock.Any(), f.ts, nasIP).Return(nil),
					f.ts.EXPECT().Commit().Return(nil),
				)
			},
			args: args{
				ctx:         context.Background(),
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
				f.ts.EXPECT().Start(gomock.Any()).Return(nil).Times(2)
				f.ts.EXPECT().Rollback().Return(nil).Times(2)

				gomock.InOrder(
					f.ri.MockRepository.FlowBatch.EXPECT().
						LoadCommittedFileNames(gomock.Any(), f.ts, nasIP).Return(committedFileNames, nil),
					f.traffic.EXPECT().NewFlowAccumulator(channelMap).
						Return(f.realTraffic.NewFlowAccumulator(channelMap)),
					f.flow.EXPECT().StreamFlow(nasIP, committedFileNames, gomock.Any()).
						DoAndReturn(streamLines(flowStr, fileNameList)),
					f.traffic.EXPECT().
						SiftTraffic(channelMap, trafficMap, sessionList).Return(chunkList, nil),
					f.ri.MockRepository.Session.EXPECT().SaveChunkList(gomock.Any(), f.ts, chunkList).Return(nil),
					f.ri.MockRepository.FlowBatch.EXPECT().
						SaveFileNames(gomock.Any(), f.ts, nasIP, fileNameList).Return(nil),
					f.ts.EXPECT().Commit().Return(nil),
					f.ri.MockRepository.Flow.EXPECT().
						RemoveOld(nasIP).Return(errors.New("диск недоступен")),
				)

				// RemoveByNasIP не ожидается: чекпоинт обязан пережить неудачную очистку
			},
			args: args{
				ctx:         context.Background(),
				nasIP:       nasIP,
				sessionList: sessionList,
				channelMap:  channelMap,
			},
		},
		{
			name: "flow без internal трафика, tmp очищается без чекпоинта",
			prepare: func(f *fields) {
				committedFileNames := map[string]bool{}
				fileNameList := []string{newFile}

				// две транзакции: загрузка чекпоинта и его удаление при очистке tmp,
				// транзакция сохранения чанков не открывается
				f.ri.SessionManager.EXPECT().CreateSession().Return(f.ts).Times(2)
				f.ts.EXPECT().Start(gomock.Any()).Return(nil).Times(2)
				f.ts.EXPECT().Rollback().Return(nil).Times(2)

				gomock.InOrder(
					f.ri.MockRepository.FlowBatch.EXPECT().
						LoadCommittedFileNames(gomock.Any(), f.ts, nasIP).Return(committedFileNames, nil),
					f.traffic.EXPECT().NewFlowAccumulator(channelMap).
						Return(f.realTraffic.NewFlowAccumulator(channelMap)),
					// обе стороны во внешней сети: строки валидны, но internal трафика нет —
					// global.ErrNoData приходит из настоящего acc.Result()
					f.flow.EXPECT().StreamFlow(nasIP, committedFileNames, gomock.Any()).
						DoAndReturn(streamLines(externalOnlyFlow, fileNameList)),
					// файлы уже в tmp: их обязательно нужно убрать, иначе они копятся
					f.ri.MockRepository.Flow.EXPECT().RemoveOld(nasIP).Return(nil),
					f.ri.MockRepository.FlowBatch.EXPECT().RemoveByNasIP(gomock.Any(), f.ts, nasIP).Return(nil),
					f.ts.EXPECT().Commit().Return(nil),
				)

				// SaveChunkList / SaveFileNames не ожидаются: сохранять нечего
			},
			args: args{
				ctx:         context.Background(),
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
				f.ts.EXPECT().Start(gomock.Any()).Return(nil).Times(2)
				f.ts.EXPECT().Rollback().Return(nil).Times(2)

				gomock.InOrder(
					f.ri.MockRepository.FlowBatch.EXPECT().
						LoadCommittedFileNames(gomock.Any(), f.ts, nasIP).Return(committedFileNames, nil),
					f.traffic.EXPECT().NewFlowAccumulator(channelMap).
						Return(f.realTraffic.NewFlowAccumulator(channelMap)),
					f.flow.EXPECT().StreamFlow(nasIP, committedFileNames, gomock.Any()).
						DoAndReturn(streamLines(flowStr, fileNameList)),
					f.traffic.EXPECT().
						SiftTraffic(channelMap, trafficMap, sessionList).Return(nil, global.ErrNoData),
					f.ri.MockRepository.Flow.EXPECT().RemoveOld(nasIP).Return(nil),
					f.ri.MockRepository.FlowBatch.EXPECT().RemoveByNasIP(gomock.Any(), f.ts, nasIP).Return(nil),
					f.ts.EXPECT().Commit().Return(nil),
				)
			},
			args: args{
				ctx:         context.Background(),
				nasIP:       nasIP,
				sessionList: sessionList,
				channelMap:  channelMap,
			},
		},
		{
			name: "flow не распознан, tmp не трогаем",
			prepare: func(f *fields) {
				committedFileNames := map[string]bool{}
				fileNameList := []string{newFile}

				// одна транзакция: только загрузка чекпоинта.
				// очистка tmp не вызывается — flow должен остаться на диске
				f.ri.SessionManager.EXPECT().CreateSession().Return(f.ts).Times(1)
				f.ts.EXPECT().Start(gomock.Any()).Return(nil).Times(1)
				f.ts.EXPECT().Rollback().Return(nil).Times(1)

				gomock.InOrder(
					f.ri.MockRepository.FlowBatch.EXPECT().
						LoadCommittedFileNames(gomock.Any(), f.ts, nasIP).Return(committedFileNames, nil),
					f.traffic.EXPECT().NewFlowAccumulator(channelMap).
						Return(f.realTraffic.NewFlowAccumulator(channelMap)),
					// ни одной валидной строки — global.ErrInternalError приходит
					// из настоящего acc.Result()
					f.flow.EXPECT().StreamFlow(nasIP, committedFileNames, gomock.Any()).
						DoAndReturn(streamLines(brokenFlow, fileNameList)),
				)

				// SiftTraffic / RemoveOld / RemoveByNasIP не ожидаются
			},
			args: args{
				ctx:         context.Background(),
				nasIP:       nasIP,
				sessionList: sessionList,
				channelMap:  channelMap,
			},
		},
		{
			name: "ошибка StreamFlow в середине потока, обработка останавливается",
			prepare: func(f *fields) {
				committedFileNames := map[string]bool{}

				// одна транзакция: только загрузка чекпоинта.
				// StreamFlow падает в середине потока — до SiftTraffic и
				// записи чекпоинта дело не доходит
				f.ri.SessionManager.EXPECT().CreateSession().Return(f.ts).Times(1)
				f.ts.EXPECT().Start(gomock.Any()).Return(nil).Times(1)
				f.ts.EXPECT().Rollback().Return(nil).Times(1)

				gomock.InOrder(
					f.ri.MockRepository.FlowBatch.EXPECT().
						LoadCommittedFileNames(gomock.Any(), f.ts, nasIP).Return(committedFileNames, nil),
					f.traffic.EXPECT().NewFlowAccumulator(channelMap).
						Return(f.realTraffic.NewFlowAccumulator(channelMap)),
					f.flow.EXPECT().StreamFlow(nasIP, committedFileNames, gomock.Any()).
						DoAndReturn(func(_ string, _ map[string]bool, onLine func(line string) error) (
							[]string, int, error) {
							lines := strings.Split(flowStr, "\n")

							// пара строк успешно учтена аккумулятором,
							// затем поток обрывается ошибкой ввода-вывода
							if err := onLine(lines[0]); err != nil {
								return nil, 0, err
							}
							if err := onLine(lines[1]); err != nil {
								return nil, 0, err
							}

							return nil, 0, errors.New("ошибка чтения flow в середине потока")
						}),
				)

				// SiftTraffic / SaveChunkList / RemoveOld / RemoveByNasIP не ожидаются:
				// обработка nas_ip прерывается сразу после ошибки StreamFlow
			},
			args: args{
				ctx:         context.Background(),
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
				ri:      rimport.NewTestRepositoryImports(ctrl),
				ts:      transaction.NewMockSession(ctrl),
				flow:    usecase.NewMockFlowStreamer(ctrl),
				session: usecase.NewMockSessionLoader(ctrl),
				channel: usecase.NewMockChannelLoader(ctrl),
				traffic: usecase.NewMockTrafficProcessor(ctrl),
			}
			f.realTraffic = newTestTrafficUsecase(t, f.ri.RepositoryImports())
			if tt.prepare != nil {
				tt.prepare(&f)
			}

			u := usecase.NewAggregatorUsecase(testLogger, f.ri.RepositoryImports(),
				usecase.AggregatorDeps{
					Flow:    f.flow,
					Session: f.session,
					Channel: f.channel,
					Traffic: f.traffic,
				}, metrics.Nop(), tracing.Nop())

			u.Aggregate(tt.args.ctx, tt.args.nasIP, tt.args.sessionList, tt.args.channelMap)
		})
	}
}

// scrapeMetrics снимает состояние метрик через http-хендлер
func scrapeMetrics(t *testing.T, m *metrics.Metrics) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	m.Handler().ServeHTTP(rec, req)

	return rec.Body.String()
}

// requireMetricLine проверяет наличие строки среди снятых серий
func requireMetricLine(t *testing.T, body, want string) {
	t.Helper()

	for _, line := range strings.Split(body, "\n") {
		if strings.TrimSpace(line) == want {
			return
		}
	}

	t.Errorf("в выводе метрик нет строки %q", want)
}

func TestAggregateMetrics(t *testing.T) {
	type fields struct {
		ri      rimport.TestRepositoryImports
		ts      *transaction.MockSession
		flow    *usecase.MockFlowStreamer
		session *usecase.MockSessionLoader
		channel *usecase.MockChannelLoader
		traffic *usecase.MockTrafficProcessor
		// realTraffic источник живого FlowAccumulator для мока TrafficProcessor
		realTraffic *usecase.TrafficUsecase
	}

	const (
		nasIP   = "127.0.0.0"
		ip1     = "127.0.0.1"
		sessID  = 1
		newFile = "ft-01.01.2026-00:05:00"
	)

	flowStr := "132,127.0.0.1,127.0.0.2"
	brokenFlow := "notanumber,127.0.0.1,127.0.0.2"

	channelMap := map[channel.ID]bool{
		channel.Internal: true,
		channel.External: false,
	}
	// одна строка internal→internal: получателю download, отправителю upload
	trafficMap := map[session.IP]map[channel.ID]traffic.Traffic{
		ip1: {
			channel.Internal: {Download: 132},
		},
		"127.0.0.2": {
			channel.Internal: {Upload: 132},
		},
	}
	sessionList := []session.OnlineSession{
		{SessID: sessID, NasIP: nasIP, IP: ip1},
	}
	chunkList := []session.Chunk{
		{SessID: sessID, ChannelID: int(channel.Internal), Download: 64, Upload: 2},
	}

	tests := []struct {
		name    string
		prepare func(f *fields)
		want    []string
	}{
		{
			name: "успешный путь пишет ok, чанки и трафик",
			prepare: func(f *fields) {
				committedFileNames := map[string]bool{}
				fileNameList := []string{newFile}

				f.ri.SessionManager.EXPECT().CreateSession().Return(f.ts).Times(3)
				f.ts.EXPECT().Start(gomock.Any()).Return(nil).Times(3)
				f.ts.EXPECT().Rollback().Return(nil).Times(3)

				gomock.InOrder(
					f.ri.MockRepository.FlowBatch.EXPECT().
						LoadCommittedFileNames(gomock.Any(), f.ts, nasIP).Return(committedFileNames, nil),
					f.traffic.EXPECT().NewFlowAccumulator(channelMap).
						Return(f.realTraffic.NewFlowAccumulator(channelMap)),
					f.flow.EXPECT().StreamFlow(nasIP, committedFileNames, gomock.Any()).
						DoAndReturn(streamLines(flowStr, fileNameList)),
					f.traffic.EXPECT().
						SiftTraffic(channelMap, trafficMap, sessionList).Return(chunkList, nil),
					f.ri.MockRepository.Session.EXPECT().SaveChunkList(gomock.Any(), f.ts, chunkList).Return(nil),
					f.ri.MockRepository.FlowBatch.EXPECT().
						SaveFileNames(gomock.Any(), f.ts, nasIP, fileNameList).Return(nil),
					f.ts.EXPECT().Commit().Return(nil),
					f.ri.MockRepository.Flow.EXPECT().RemoveOld(nasIP).Return(nil),
					f.ri.MockRepository.FlowBatch.EXPECT().RemoveByNasIP(gomock.Any(), f.ts, nasIP).Return(nil),
					f.ts.EXPECT().Commit().Return(nil),
				)
			},
			want: []string{
				`aggregator_nas_processed_total{result="ok"} 1`,
				`aggregator_chunks_saved_total 1`,
				`aggregator_accounted_traffic_bytes_total{direction="download"} 64`,
				`aggregator_accounted_traffic_bytes_total{direction="upload"} 2`,
				`aggregator_nas_phase_duration_seconds_count{phase="stream_flow"} 1`,
				`aggregator_nas_phase_duration_seconds_count{phase="save_chunks"} 1`,
				`aggregator_flow_size_bytes_count 1`,
			},
		},
		{
			name: "нераспознанный flow пишет unrecognized и ошибку этапа parse",
			prepare: func(f *fields) {
				committedFileNames := map[string]bool{}
				fileNameList := []string{newFile}

				f.ri.SessionManager.EXPECT().CreateSession().Return(f.ts).Times(1)
				f.ts.EXPECT().Start(gomock.Any()).Return(nil).Times(1)
				f.ts.EXPECT().Rollback().Return(nil).Times(1)

				gomock.InOrder(
					f.ri.MockRepository.FlowBatch.EXPECT().
						LoadCommittedFileNames(gomock.Any(), f.ts, nasIP).Return(committedFileNames, nil),
					f.traffic.EXPECT().NewFlowAccumulator(channelMap).
						Return(f.realTraffic.NewFlowAccumulator(channelMap)),
					f.flow.EXPECT().StreamFlow(nasIP, committedFileNames, gomock.Any()).
						DoAndReturn(streamLines(brokenFlow, fileNameList)),
				)
			},
			want: []string{
				`aggregator_nas_processed_total{result="unrecognized"} 1`,
				`aggregator_nas_errors_total{stage="parse"} 1`,
				`aggregator_nas_processed_total{result="ok"} 0`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			f := fields{
				ri:      rimport.NewTestRepositoryImports(ctrl),
				ts:      transaction.NewMockSession(ctrl),
				flow:    usecase.NewMockFlowStreamer(ctrl),
				session: usecase.NewMockSessionLoader(ctrl),
				channel: usecase.NewMockChannelLoader(ctrl),
				traffic: usecase.NewMockTrafficProcessor(ctrl),
			}
			f.realTraffic = newTestTrafficUsecase(t, f.ri.RepositoryImports())
			tt.prepare(&f)

			m := metrics.New(testLogger, "test", nil)

			u := usecase.NewAggregatorUsecase(testLogger, f.ri.RepositoryImports(),
				usecase.AggregatorDeps{
					Flow:    f.flow,
					Session: f.session,
					Channel: f.channel,
					Traffic: f.traffic,
				}, m, tracing.Nop())

			u.Aggregate(context.Background(), nasIP, sessionList, channelMap)

			body := scrapeMetrics(t, m)

			for _, want := range tt.want {
				requireMetricLine(t, body, want)
			}
		})
	}
}
