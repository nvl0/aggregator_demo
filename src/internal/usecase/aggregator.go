package usecase

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"aggregator/src/bimport"
	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/entity/session"
	"aggregator/src/rimport"
	"aggregator/src/tools/dump"
	"aggregator/src/tools/flowgen"
	"aggregator/src/tools/measure"
	"aggregator/src/tools/metrics"
	"aggregator/src/tools/workerpool"

	"github.com/sirupsen/logrus"
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
	log     *logrus.Logger
	// poolSize размер пула воркеров агрегации
	poolSize int
	//
	rimport.RepositoryImports
	*bimport.BridgeImports
}

func NewAggregatorUsecase(
	log *logrus.Logger,
	ri rimport.RepositoryImports,
	bi *bimport.BridgeImports,
	m *metrics.Metrics,
) *AggregatorUsecase {
	writer := measure.NewLogrusWriter(log)
	// msr, а не m: имя m занято метриками
	msr := measure.NewMeasure(writer)

	u := &AggregatorUsecase{
		measure:           msr,
		metrics:           m,
		log:               log,
		poolSize:          ri.Config.WorkerPoolSize(),
		RepositoryImports: ri,
		BridgeImports:     bi,
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
			u.log.Debugln("ожидаемый результат", dump.Struct(expectedChunk))
		} else {
			u.log.Debugln("не удалось загрузить список nas_ip директорий, ошибка", err)
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
		u.loadChannelMap(chanChan)
		u.metrics.ObservePhase(metrics.PhaseLoadChannels, time.Since(phaseStart))
	}()
	// получение мапки сессий
	go func() {
		defer loaders.Done()

		phaseStart := time.Now()
		u.loadOnlineSessionMap(sessChan)
		u.metrics.ObservePhase(metrics.PhaseLoadSessions, time.Since(phaseStart))
	}()

	u.measure.Start("получение списка директорий")
	dirsStart := time.Now()
	dirList, err := u.Repository.Flow.ReadFlowDirNames()
	// фаза пишется и на неуспехе
	u.metrics.ObservePhase(metrics.PhaseReadDirs, time.Since(dirsStart))

	if err != nil {
		u.metrics.IncCycleError(metrics.CycleErrDirsRead)
		u.log.Debugln("не удалось загрузить список nas_ip директорий, ошибка", err)
		loaders.Wait() // не оставляем загрузочные горутины висеть

		return
	}
	u.measure.Stop("получение списка директорий")
	u.log.Debugf("количество директорий %d", len(dirList))

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
		u.log.WithFields(logrus.Fields{
			"recovered": recovered,
			"stack":     string(debug.Stack()),
		}).Errorln("паника воркера агрегации, обработка nas_ip прервана")
	}))

	// название директории совпадает с session.NasIP
	for _, nasIP := range dirList {
		// если директория не совпадет с session.NasIP
		// то обработка директории будет отброшена
		sessionList, exists := sessionMap[nasIP]
		if !exists {
			u.log.WithField(logFieldNasIP, nasIP).Debugf("nas_ip %s отсутствует в бд", nasIP)
			continue
		}

		// копия nasIP для явности; sessionList копировать не нужно, она объявлена внутри тела цикла

		// если контекст отменился во время ожидания свободного слота,
		// рассылка оставшихся nas_ip прекращается
		if !pool.Go(ctx, func() {
			u.Bridge.Aggregator.Aggregate(nasIP, sessionList, channelMap)
		}) {
			u.log.Debugln("контекст отменен, рассылка оставшихся nas_ip прекращена")
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
func (u *AggregatorUsecase) loadChannelMap(chanChan chan<- map[channel.ID]bool) {
	defer close(chanChan)

	ts := u.SessionManager.CreateSession()
	if err := ts.Start(); err != nil {
		u.log.Errorln("не удалось открыть транзакцию, ошибка", err)
		return
	}
	defer func() { _ = ts.Rollback() }()

	chanLogName := "получение мапки каналов"
	u.measure.Start(chanLogName)
	defer u.measure.Stop(chanLogName)

	channelMap, err := u.Bridge.Channel.LoadChannelMap(ts)
	if err != nil {
		u.log.Errorln("не удалось загрузить мапку каналов, ошибка", err)
		return
	}

	chanChan <- channelMap
}

// loadOnlineSessionMap загрузка онлайн сессий
func (u *AggregatorUsecase) loadOnlineSessionMap(sessChan chan<- map[session.NasIP][]session.OnlineSession) {
	defer close(sessChan)

	ts := u.SessionManager.CreateSession()
	if err := ts.Start(); err != nil {
		u.log.Errorln("не удалось открыть транзакцию, ошибка", err)
		return
	}
	defer func() { _ = ts.Rollback() }()

	sessLogName := "получение мапки онлайн сессий"
	u.measure.Start(sessLogName)
	defer u.measure.Stop(sessLogName)

	sessionMap, err := u.Bridge.Session.LoadOnlineSessionMap(ts)
	if err != nil {
		u.log.Errorln("не удалось загрузить мапку онлайн сессий, ошибка", err)
		return
	}

	sessChan <- sessionMap
}

// Aggregate агрегация траффика
func (u *AggregatorUsecase) Aggregate(
	nasIP string,
	sessionList []session.OnlineSession,
	channelMap map[channel.ID]bool,
) {
	writer := measure.NewLogrusWriter(u.log)
	m := measure.NewMeasure(writer)

	lf := logrus.Fields{
		logFieldNasIP: nasIP,
	}

	u.log.WithFields(lf).Debugf("количество сессий онлайн %d", len(sessionList))

	// имена flow файлов, чанки которых уже закоммичены в одном из предыдущих циклов
	committedFileNames, err := u.loadCommittedFileNames(nasIP)
	if err != nil {
		u.nasFailed(metrics.NASStageCheckpoint)

		return
	}

	m.Start(fmt.Sprintf("%s подготовка flow", nasIP))
	// prepareFlow фиксирует длительность фазы, в том числе на ошибке
	flow, fileNameList, err := u.prepareFlow(nasIP, committedFileNames)
	if err != nil {
		u.nasFailed(metrics.NASStagePrepare)

		return
	}
	m.Stop(fmt.Sprintf("%s подготовка flow", nasIP))
	u.log.WithFields(lf).Debugf("размер flow %d", len([]rune(flow)))

	// метрика берет длину в байтах, а не в рунах
	u.metrics.ObserveFlowSize(len(flow))

	// весь tmp состоит из уже закоммиченных файлов: предыдущий цикл упал
	// между коммитом чанков и очисткой tmp. Считать нечего, нужно лишь завершить очистку
	if !hasNewFile(fileNameList, committedFileNames) {
		u.log.WithFields(lf).Debugln("новых flow файлов нет, повторная очистка tmp")
		u.removeOldFlow(nasIP)
		m.Result()

		u.metrics.IncNAS(metrics.NASResultNoNew)

		return
	}

	parseFlowLogName := fmt.Sprintf("%s парсинг flow, подсчет трафика", nasIP)
	m.Start(parseFlowLogName)
	parseStart := time.Now()
	trafficMap, err := u.Bridge.Traffic.ParseFlow(channelMap, flow)
	u.metrics.ObserveNASPhase(metrics.NASPhaseParseFlow, time.Since(parseStart))

	switch {
	case errors.Is(err, global.ErrNoData):
		// flow распарсен, но учитываемого (internal) трафика в нем нет — только external.
		// считать нечего, файлы нужно убрать из tmp, иначе они копятся
		// и перечитываются на каждом цикле
		u.log.WithFields(lf).Warnln("во flow нет internal трафика, очистка tmp")
		u.removeOldFlow(nasIP)
		m.Result()

		u.metrics.IncNAS(metrics.NASResultNoInternal)

		return
	case err != nil:
		// flow не распознан (дрейф формата, сбой классификации по internal).
		// файлы намеренно оставляем в tmp для следующего цикла и разбора
		u.log.WithFields(lf).Warnln("flow не дал трафика и не распознан, файлы оставлены в tmp, ошибка", err)
		u.metrics.IncNASError(metrics.NASStageParse)
		u.metrics.IncNAS(metrics.NASResultUnrecognized)

		return
	}
	m.Stop(parseFlowLogName)
	u.log.WithFields(lf).Debugf("количество трафика %d", len(trafficMap))

	siftTrafficLogName := fmt.Sprintf("%s привязка трафика к сессии", nasIP)
	m.Start(siftTrafficLogName)
	siftStart := time.Now()
	chunkList, err := u.Bridge.Traffic.SiftTraffic(channelMap, trafficMap, sessionList)
	u.metrics.ObserveNASPhase(metrics.NASPhaseSiftTraffic, time.Since(siftStart))

	switch {
	case errors.Is(err, global.ErrNoData):
		// просеивать нечего (например, пустой список сессий), но flow уже в tmp —
		// убираем, чтобы файлы не накапливались и не перечитывались каждый цикл
		u.log.WithFields(lf).Warnln("нет данных для просеивания трафика, очистка tmp")
		u.removeOldFlow(nasIP)
		m.Result()

		u.nasFailed(metrics.NASStageSift)

		return
	case err != nil:
		u.nasFailed(metrics.NASStageSift)

		return
	}
	m.Stop(siftTrafficLogName)
	u.log.WithFields(lf).Debugf("количество чанков %d", len(chunkList))
	u.log.Debugln("актуальный результат", dump.Struct(chunkList))

	u.accountTraffic(chunkList)

	saveChunkListLogName := fmt.Sprintf("%s сохранение чанков и чекпоинта в бд", nasIP)
	m.Start(saveChunkListLogName)

	if err = u.commitChunks(nasIP, chunkList, fileNameList); err != nil {
		return
	}
	m.Stop(saveChunkListLogName)

	u.metrics.AddChunksSaved(len(chunkList))
	u.metrics.IncNAS(metrics.NASResultOK)

	u.removeOldFlow(nasIP)

	m.Result()
}

// nasFailed фиксация неуспешной обработки nas_ip на этапе stage
func (u *AggregatorUsecase) nasFailed(stage metrics.NASStage) {
	u.metrics.IncNASError(stage)
	u.metrics.IncNAS(metrics.NASResultError)
}

// prepareFlow подготовка flow с фиксацией длительности фазы.
// фаза пишется и на ошибке
func (u *AggregatorUsecase) prepareFlow(
	nasIP string,
	committedFileNames map[string]bool,
) (flow string, fileNameList []string, err error) {
	start := time.Now()
	flow, fileNameList, err = u.Bridge.Flow.PrepareFlow(nasIP, committedFileNames)
	u.metrics.ObserveNASPhase(metrics.NASPhasePrepareFlow, time.Since(start))

	return flow, fileNameList, err
}

// commitChunks сохранение чанков с фиксацией метрик этапа
func (u *AggregatorUsecase) commitChunks(
	nasIP string,
	chunkList []session.Chunk,
	fileNameList []string,
) error {
	start := time.Now()
	err := u.saveChunkListWithCheckpoint(nasIP, chunkList, fileNameList)
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
func (u *AggregatorUsecase) loadCommittedFileNames(nasIP string) (fileNameSet map[string]bool, err error) {
	lf := logrus.Fields{
		logFieldNasIP: nasIP,
	}

	ts := u.SessionManager.CreateSession()
	if err = ts.Start(); err != nil {
		u.log.Errorln("не удалось открыть транзакцию, ошибка", err)
		return fileNameSet, err
	}
	defer func() { _ = ts.Rollback() }()

	if fileNameSet, err = u.Repository.FlowBatch.LoadCommittedFileNames(ts, nasIP); err != nil {
		u.log.WithFields(lf).Errorln("не удалось загрузить чекпоинт flow файлов, ошибка", err)
		return fileNameSet, err
	}

	return fileNameSet, err
}

// saveChunkListWithCheckpoint сохранение чанков сессии и чекпоинта flow файлов
// в одной транзакции: чекпоинт пишется в той же транзакции, что и чанки, и содержит
// весь список файлов из tmp, а не только новые. Если очистка tmp не удастся или процесс
// упадет сразу после коммита, следующий цикл не посчитает эти файлы повторно
func (u *AggregatorUsecase) saveChunkListWithCheckpoint(
	nasIP string,
	chunkList []session.Chunk,
	fileNameList []string,
) error {
	lf := logrus.Fields{
		logFieldNasIP: nasIP,
	}

	ts := u.SessionManager.CreateSession()
	if err := ts.Start(); err != nil {
		u.log.Errorln("не удалось открыть транзакцию, ошибка", err)
		return err
	}
	defer func() { _ = ts.Rollback() }()

	if err := u.Repository.Session.SaveChunkList(ts, chunkList); err != nil {
		u.log.WithFields(lf).Errorln("не удалось сохранить чанки, ошибка", err)
		return err
	}

	if err := u.Repository.FlowBatch.SaveFileNames(ts, nasIP, fileNameList); err != nil {
		u.log.WithFields(lf).Errorln("не удалось сохранить чекпоинт flow файлов, ошибка", err)
		return err
	}

	if err := ts.Commit(); err != nil {
		u.log.Errorln("не удалось закрыть транзакцию, ошибка", err)
		return err
	}

	return nil
}

// removeOldFlow удаление обработанного flow вместе с чекпоинтом.
// Если удалить файлы не удалось, записи чекпоинта намеренно остаются в бд:
// именно они защищают от повторного подсчета этих файлов на следующем цикле
func (u *AggregatorUsecase) removeOldFlow(nasIP string) {
	lf := logrus.Fields{
		logFieldNasIP: nasIP,
	}

	if err := u.Repository.Flow.RemoveOld(nasIP); err != nil {
		u.log.WithFields(lf).Errorln("не удалось удалить старый flow, ошибка", err)
		return
	}

	ts := u.SessionManager.CreateSession()
	if err := ts.Start(); err != nil {
		u.log.Errorln("не удалось открыть транзакцию, ошибка", err)
		return
	}
	defer func() { _ = ts.Rollback() }()

	if err := u.Repository.FlowBatch.RemoveByNasIP(ts, nasIP); err != nil {
		u.log.WithFields(lf).Errorln("не удалось удалить чекпоинт flow файлов, ошибка", err)
		return
	}

	if err := ts.Commit(); err != nil {
		u.log.Errorln("не удалось закрыть транзакцию, ошибка", err)
	}
}
