package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/entity/session"
	"aggregator/src/rimport"
	"aggregator/src/tools/dump"
	"aggregator/src/tools/flowgen"
	"aggregator/src/tools/measure"
	"aggregator/src/tools/metrics"
	"aggregator/src/tools/workerpool"
)

const (
	// logFieldNasIP ключ поля nas_ip в структурных логах
	logFieldNasIP = "nas_ip"
	// loaderCount число стартовых загрузочных горутин: мапка каналов и мапка онлайн сессий
	loaderCount = 2
)

type AggregatorUsecase struct {
	measure measure.Measure
	metrics *metrics.Metrics
	log     *slog.Logger
	// poolSize размер пула воркеров агрегации
	poolSize int
	// прямые зависимости на соседние usecase вместо индирекции через bridge
	flow    FlowStreamer
	session SessionLoader
	channel ChannelLoader
	traffic TrafficProcessor
	//
	rimport.RepositoryImports
}

// AggregatorDeps зависимости агрегатора на соседние usecase.
// Структура-параметр вместо четырех позиционных аргументов конструктора
type AggregatorDeps struct {
	Flow    FlowStreamer
	Session SessionLoader
	Channel ChannelLoader
	Traffic TrafficProcessor
}

func NewAggregatorUsecase(
	log *slog.Logger,
	ri rimport.RepositoryImports,
	deps AggregatorDeps,
	m *metrics.Metrics,
) *AggregatorUsecase {
	writer := measure.NewSlogWriter(log)
	// msr, а не m: имя m занято метриками
	msr := measure.NewMeasure(writer)

	u := &AggregatorUsecase{
		measure:           msr,
		metrics:           m,
		log:               log,
		poolSize:          ri.Config.WorkerPoolSize(),
		flow:              deps.Flow,
		session:           deps.Session,
		channel:           deps.Channel,
		traffic:           deps.Traffic,
		RepositoryImports: ri,
	}

	m.SetPoolSize(u.poolSize)

	return u
}

var fgen = os.Getenv("FLOWGEN") == "true"

// Start запуск агрегатора
func (u *AggregatorUsecase) Start(ctx context.Context) {
	cycleStart := time.Now()
	cycleOK := false

	u.metrics.SetCycleInProgress(true)
	defer u.observeCycle(cycleStart, &cycleOK)

	// генерация flow
	if fgen {
		if down, up, err := flowgen.Generate(flowgen.Params{
			NasIP:    flowgen.DefaultNasIP,
			ClientIP: flowgen.DefaultClientIP,
			FlowDir:  flowgen.DefaultFlowDir,
		}); err == nil {
			expectedChunk := session.Chunk{
				SessID:    1,
				ChannelID: int(channel.External),
				Download:  down,
				Upload:    up,
			}
			u.log.DebugContext(ctx, "ожидаемый результат", "dump", dump.Struct(expectedChunk))
		} else {
			u.log.DebugContext(ctx, "не удалось загрузить список nas_ip директорий, ошибка", "error", err)
		}
	}

	// буфер на 1: отправка результата в горутине не блокируется, даже если
	// вызывающий вышел по ошибке и не стал читать канал
	chanChan := make(chan map[channel.ID]bool, 1)
	sessChan := make(chan map[session.NasIP][]session.OnlineSession, 1)

	var loaders sync.WaitGroup
	loaders.Add(loaderCount)
	// получение мапки каналов
	go func() {
		defer loaders.Done()

		phaseStart := time.Now()
		u.loadChannelMap(ctx, chanChan)
		u.metrics.ObservePhase(metrics.PhaseLoadChannels, time.Since(phaseStart))
	}()
	// получение мапки сессий
	go func() {
		defer loaders.Done()

		phaseStart := time.Now()
		u.loadOnlineSessionMap(ctx, sessChan)
		u.metrics.ObservePhase(metrics.PhaseLoadSessions, time.Since(phaseStart))
	}()

	u.measure.Start("получение списка директорий")
	dirsStart := time.Now()
	dirList, err := u.Repository.Flow.ReadFlowDirNames()
	// фаза пишется и на неуспехе
	u.metrics.ObservePhase(metrics.PhaseReadDirs, time.Since(dirsStart))

	if err != nil {
		u.metrics.IncCycleError(metrics.CycleErrDirsRead)
		u.log.DebugContext(ctx, "не удалось загрузить список nas_ip директорий, ошибка", "error", err)
		loaders.Wait() // не оставляем загрузочные горутины висеть

		return
	}
	u.measure.Stop("получение списка директорий")
	u.log.DebugContext(ctx, "количество директорий", "count", len(dirList))

	u.metrics.SetNASDiscovered(len(dirList))

	channelMap := <-chanChan
	sessionMap := <-sessChan
	// обе загрузочные горутины докатывают defer'ы (Rollback транзакций, measure.Stop)
	// до того, как метод пойдет дальше или вернется
	loaders.Wait()

	if channelMap == nil || sessionMap == nil {
		u.metrics.IncCycleError(metrics.CycleErrNilMaps)

		return
	}

	cycleOK = true

	u.measure.Result()

	pool := workerpool.New(u.poolSize, workerpool.WithOnPanic(func(recovered any) {
		u.metrics.IncWorkerPanic()
		u.log.ErrorContext(ctx, "паника воркера агрегации, обработка nas_ip прервана",
			"recovered", recovered, "stack", string(debug.Stack()))
	}))

	// название директории совпадает с session.NasIP
	for _, nasIP := range dirList {
		// если директория не совпадет с session.NasIP
		// то обработка директории будет отброшена
		sessionList, exists := sessionMap[nasIP]
		if !exists {
			u.log.DebugContext(ctx, "nas_ip отсутствует в бд", logFieldNasIP, nasIP)
			continue
		}

		// копия nasIP для явности; sessionList копировать не нужно, она объявлена внутри тела цикла

		// если контекст отменился во время ожидания свободного слота,
		// рассылка оставшихся nas_ip прекращается
		if !pool.Go(ctx, func() {
			u.Aggregate(ctx, nasIP, sessionList, channelMap)
		}) {
			u.log.DebugContext(ctx, "контекст отменен, рассылка оставшихся nas_ip прекращена")
			break
		}
	}

	// уже стартовавшие воркеры не прерываются на середине
	pool.Wait()
}

// observeCycle фиксация метрик завершившегося цикла.
// cycleOK передается указателем: значение выставляется уже после
// того, как defer с вызовом этого метода объявлен
func (u *AggregatorUsecase) observeCycle(start time.Time, cycleOK *bool) {
	elapsed := time.Since(start)

	u.metrics.ObserveCycle(elapsed)
	u.metrics.SetLastCycleDuration(elapsed)
	u.metrics.SetCycleInProgress(false)

	if *cycleOK {
		u.metrics.SetLastSuccess()
	}
}

// loadChannelMap загрузка каналов
func (u *AggregatorUsecase) loadChannelMap(ctx context.Context, chanChan chan<- map[channel.ID]bool) {
	defer close(chanChan)

	ts := u.SessionManager.CreateSession()
	if err := ts.Start(ctx); err != nil {
		u.log.ErrorContext(ctx, "не удалось открыть транзакцию, ошибка", "error", err)
		return
	}
	defer func() { _ = ts.Rollback() }()

	chanLogName := "получение мапки каналов"
	u.measure.Start(chanLogName)
	defer u.measure.Stop(chanLogName)

	channelMap, err := u.channel.LoadChannelMap(ctx, ts)
	if err != nil {
		u.log.ErrorContext(ctx, "не удалось загрузить мапку каналов, ошибка", "error", err)
		return
	}

	chanChan <- channelMap
}

// loadOnlineSessionMap загрузка онлайн сессий
func (u *AggregatorUsecase) loadOnlineSessionMap(
	ctx context.Context,
	sessChan chan<- map[session.NasIP][]session.OnlineSession,
) {
	defer close(sessChan)

	ts := u.SessionManager.CreateSession()
	if err := ts.Start(ctx); err != nil {
		u.log.ErrorContext(ctx, "не удалось открыть транзакцию, ошибка", "error", err)
		return
	}
	defer func() { _ = ts.Rollback() }()

	sessLogName := "получение мапки онлайн сессий"
	u.measure.Start(sessLogName)
	defer u.measure.Stop(sessLogName)

	sessionMap, err := u.session.LoadOnlineSessionMap(ctx, ts)
	if err != nil {
		u.log.ErrorContext(ctx, "не удалось загрузить мапку онлайн сессий, ошибка", "error", err)
		return
	}

	sessChan <- sessionMap
}

// Aggregate агрегация траффика
func (u *AggregatorUsecase) Aggregate(
	ctx context.Context,
	nasIP string,
	sessionList []session.OnlineSession,
	channelMap map[channel.ID]bool,
) {
	writer := measure.NewSlogWriter(u.log)
	m := measure.NewMeasure(writer)

	u.log.DebugContext(ctx, "количество сессий онлайн", "count", len(sessionList), logFieldNasIP, nasIP)

	// имена flow файлов, чанки которых уже закоммичены в одном из предыдущих циклов
	committedFileNames, err := u.loadCommittedFileNames(ctx, nasIP)
	if err != nil {
		u.nasFailed(metrics.NASStageCheckpoint)

		return
	}

	acc := u.traffic.NewFlowAccumulator(channelMap)

	streamLogName := fmt.Sprintf("%s потоковый разбор flow", nasIP)
	m.Start(streamLogName)
	// streamFlow фиксирует длительность фазы, в том числе на ошибке
	fileNameList, flowSize, err := u.streamFlow(nasIP, committedFileNames, acc.AccumulateLine)
	if err != nil {
		u.nasFailed(metrics.NASStagePrepare)

		return
	}
	m.Stop(streamLogName)

	u.log.DebugContext(ctx, "размер flow", "size", flowSize, logFieldNasIP, nasIP)
	u.metrics.ObserveFlowSize(flowSize)

	// весь tmp состоит из уже закоммиченных файлов: предыдущий цикл упал
	// между коммитом чанков и очисткой tmp. Считать нечего, нужно лишь завершить очистку
	if !hasNewFile(fileNameList, committedFileNames) {
		u.log.DebugContext(ctx, "новых flow файлов нет, повторная очистка tmp", logFieldNasIP, nasIP)
		u.removeOldFlow(ctx, nasIP)
		m.Result()

		u.metrics.IncNAS(metrics.NASResultNoNew)

		return
	}

	trafficMap, err := acc.Result()

	switch {
	case errors.Is(err, global.ErrNoData):
		// flow распарсен, но учитываемого (internal) трафика в нем нет — только external.
		// считать нечего, файлы нужно убрать из tmp, иначе они копятся
		// и перечитываются на каждом цикле
		u.log.WarnContext(ctx, "во flow нет internal трафика, очистка tmp", logFieldNasIP, nasIP)
		u.removeOldFlow(ctx, nasIP)
		m.Result()

		u.metrics.IncNAS(metrics.NASResultNoInternal)

		return
	case err != nil:
		// flow не распознан (дрейф формата, сбой классификации по internal).
		// файлы намеренно оставляем в tmp для следующего цикла и разбора
		u.log.WarnContext(ctx, "flow не дал трафика и не распознан, файлы оставлены в tmp, ошибка",
			"error", err, logFieldNasIP, nasIP)
		u.metrics.IncNASError(metrics.NASStageParse)
		u.metrics.IncNAS(metrics.NASResultUnrecognized)

		return
	}
	u.log.DebugContext(ctx, "количество трафика", "count", len(trafficMap), logFieldNasIP, nasIP)

	siftTrafficLogName := fmt.Sprintf("%s привязка трафика к сессии", nasIP)
	m.Start(siftTrafficLogName)
	siftStart := time.Now()
	chunkList, err := u.traffic.SiftTraffic(channelMap, trafficMap, sessionList)
	u.metrics.ObserveNASPhase(metrics.NASPhaseSiftTraffic, time.Since(siftStart))

	switch {
	case errors.Is(err, global.ErrNoData):
		// просеивать нечего (например, пустой список сессий), но flow уже в tmp —
		// убираем, чтобы файлы не накапливались и не перечитывались каждый цикл
		u.log.WarnContext(ctx, "нет данных для просеивания трафика, очистка tmp", logFieldNasIP, nasIP)
		u.removeOldFlow(ctx, nasIP)
		m.Result()

		u.nasFailed(metrics.NASStageSift)

		return
	case err != nil:
		u.nasFailed(metrics.NASStageSift)

		return
	}
	m.Stop(siftTrafficLogName)
	u.log.DebugContext(ctx, "количество чанков", "count", len(chunkList), logFieldNasIP, nasIP)
	u.log.DebugContext(ctx, "актуальный результат", "dump", dump.Struct(chunkList))

	u.accountTraffic(chunkList)

	saveChunkListLogName := fmt.Sprintf("%s сохранение чанков и чекпоинта в бд", nasIP)
	m.Start(saveChunkListLogName)

	if err = u.commitChunks(ctx, nasIP, chunkList, fileNameList); err != nil {
		return
	}
	m.Stop(saveChunkListLogName)

	u.metrics.AddChunksSaved(len(chunkList))
	u.metrics.IncNAS(metrics.NASResultOK)

	u.removeOldFlow(ctx, nasIP)

	m.Result()
}

// nasFailed фиксация неуспешной обработки nas_ip на этапе stage
func (u *AggregatorUsecase) nasFailed(stage metrics.NASStage) {
	u.metrics.IncNASError(stage)
	u.metrics.IncNAS(metrics.NASResultError)
}

// streamFlow потоковый разбор flow с фиксацией длительности фазы.
// фаза пишется и на ошибке
func (u *AggregatorUsecase) streamFlow(
	nasIP string,
	committedFileNames map[string]bool,
	onLine func(line string) error,
) (fileNameList []string, flowSize int, err error) {
	start := time.Now()
	fileNameList, flowSize, err = u.flow.StreamFlow(nasIP, committedFileNames, onLine)
	u.metrics.ObserveNASPhase(metrics.NASPhaseStreamFlow, time.Since(start))

	return fileNameList, flowSize, err
}

// commitChunks сохранение чанков с фиксацией метрик этапа
func (u *AggregatorUsecase) commitChunks(
	ctx context.Context,
	nasIP string,
	chunkList []session.Chunk,
	fileNameList []string,
) error {
	start := time.Now()
	err := u.saveChunkListWithCheckpoint(ctx, nasIP, chunkList, fileNameList)
	u.metrics.ObserveNASPhase(metrics.NASPhaseSaveChunks, time.Since(start))

	if err != nil {
		u.nasFailed(metrics.NASStageSave)
	}

	return err
}

// accountTraffic учет объема трафика по чанкам, готовым к сохранению
func (u *AggregatorUsecase) accountTraffic(chunkList []session.Chunk) {
	var download, upload int

	for _, chunk := range chunkList {
		download += chunk.Download
		upload += chunk.Upload
	}

	u.metrics.AddAccountedTraffic(metrics.DirectionDownload, download)
	u.metrics.AddAccountedTraffic(metrics.DirectionUpload, upload)
}

// hasNewFile проверка, что среди файлов есть хотя бы один незакоммиченный
func hasNewFile(fileNameList []string, committedFileNames map[string]bool) bool {
	for _, fileName := range fileNameList {
		if !committedFileNames[fileName] {
			return true
		}
	}

	return false
}

// loadCommittedFileNames загрузка имен flow файлов, чанки которых уже закоммичены
func (u *AggregatorUsecase) loadCommittedFileNames(
	ctx context.Context,
	nasIP string,
) (fileNameSet map[string]bool, err error) {
	ts := u.SessionManager.CreateSession()
	if err = ts.Start(ctx); err != nil {
		u.log.ErrorContext(ctx, "не удалось открыть транзакцию, ошибка", "error", err)
		return fileNameSet, err
	}
	defer func() { _ = ts.Rollback() }()

	if fileNameSet, err = u.Repository.FlowBatch.LoadCommittedFileNames(ctx, ts, nasIP); err != nil {
		u.log.ErrorContext(ctx, "не удалось загрузить чекпоинт flow файлов, ошибка",
			"error", err, logFieldNasIP, nasIP)
		return fileNameSet, err
	}

	return fileNameSet, err
}

// saveChunkListWithCheckpoint сохранение чанков сессии и чекпоинта flow файлов
// в одной транзакции: чекпоинт пишется в той же транзакции, что и чанки, и содержит
// весь список файлов из tmp, а не только новые. Если очистка tmp не удастся или процесс
// упадет сразу после коммита, следующий цикл не посчитает эти файлы повторно
func (u *AggregatorUsecase) saveChunkListWithCheckpoint(
	ctx context.Context,
	nasIP string,
	chunkList []session.Chunk,
	fileNameList []string,
) error {
	ts := u.SessionManager.CreateSession()
	if err := ts.Start(ctx); err != nil {
		u.log.ErrorContext(ctx, "не удалось открыть транзакцию, ошибка", "error", err)
		return err
	}
	defer func() { _ = ts.Rollback() }()

	if err := u.Repository.Session.SaveChunkList(ctx, ts, chunkList); err != nil {
		u.log.ErrorContext(ctx, "не удалось сохранить чанки, ошибка", "error", err, logFieldNasIP, nasIP)
		return err
	}

	if err := u.Repository.FlowBatch.SaveFileNames(ctx, ts, nasIP, fileNameList); err != nil {
		u.log.ErrorContext(ctx, "не удалось сохранить чекпоинт flow файлов, ошибка",
			"error", err, logFieldNasIP, nasIP)
		return err
	}

	if err := ts.Commit(); err != nil {
		u.log.ErrorContext(ctx, "не удалось закрыть транзакцию, ошибка", "error", err)
		return err
	}

	return nil
}

// removeOldFlow удаление обработанного flow вместе с чекпоинтом.
// Если удалить файлы не удалось, записи чекпоинта намеренно остаются в бд:
// именно они защищают от повторного подсчета этих файлов на следующем цикле
func (u *AggregatorUsecase) removeOldFlow(ctx context.Context, nasIP string) {
	if err := u.Repository.Flow.RemoveOld(nasIP); err != nil {
		u.log.ErrorContext(ctx, "не удалось удалить старый flow, ошибка", "error", err, logFieldNasIP, nasIP)
		return
	}

	ts := u.SessionManager.CreateSession()
	if err := ts.Start(ctx); err != nil {
		u.log.ErrorContext(ctx, "не удалось открыть транзакцию, ошибка", "error", err)
		return
	}
	defer func() { _ = ts.Rollback() }()

	if err := u.Repository.FlowBatch.RemoveByNasIP(ctx, ts, nasIP); err != nil {
		u.log.ErrorContext(ctx, "не удалось удалить чекпоинт flow файлов, ошибка",
			"error", err, logFieldNasIP, nasIP)
		return
	}

	if err := ts.Commit(); err != nil {
		u.log.ErrorContext(ctx, "не удалось закрыть транзакцию, ошибка", "error", err)
	}
}
