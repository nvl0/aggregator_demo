package usecase

import (
	"aggregator/src/bimport"
	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/session"
	"aggregator/src/rimport"
	"aggregator/src/tools/dump"
	"aggregator/src/tools/flowgen"
	"aggregator/src/tools/measure"
	"aggregator/src/tools/workerpool"
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

type AggregatorUsecase struct {
	measure measure.Measure
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
) *AggregatorUsecase {
	writer := measure.NewLogrusWriter(log)
	m := measure.NewMeasure(writer)

	return &AggregatorUsecase{
		measure:           m,
		log:               log,
		poolSize:          ri.Config.WorkerPoolSize(),
		RepositoryImports: ri,
		BridgeImports:     bi,
	}
}

var fgen = os.Getenv("FLOWGEN") == "true"

// Start запуск агрегатора
func (u *AggregatorUsecase) Start(ctx context.Context) {
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

	chanChan := make(chan map[channel.ID]bool)
	sessChan := make(chan map[session.NasIP][]session.OnlineSession)

	// получение мапки каналов
	go u.loadChannelMap(chanChan)
	// получение мапки сессий
	go u.loadOnlineSessionMap(sessChan)

	u.measure.Start("получение списка директорий")
	dirList, err := u.Repository.Flow.ReadFlowDirNames()
	if err != nil {
		u.log.Debugln("не удалось загрузить список nas_ip директорий, ошибка", err)
		return
	}
	u.measure.Stop("получение списка директорий")
	u.log.Debugf("количество директорий %d", len(dirList))

	channelMap := <-chanChan
	if channelMap == nil {
		return
	}
	sessionMap := <-sessChan
	if sessionMap == nil {
		return
	}

	u.measure.Result()

	pool := workerpool.New(u.poolSize, workerpool.WithOnPanic(func(recovered any) {
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
			u.log.WithField("nas_ip", nasIP).Debugf("nas_ip %s отсутствует в бд", nasIP)
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
		"nas_ip": nasIP,
	}

	u.log.WithFields(lf).Debugf("количество сессий онлайн %d", len(sessionList))

	// имена flow файлов, чанки которых уже закоммичены в одном из предыдущих циклов
	committedFileNames, err := u.loadCommittedFileNames(nasIP)
	if err != nil {
		return
	}

	m.Start(fmt.Sprintf("%s подготовка flow", nasIP))
	flow, fileNameList, err := u.Bridge.Flow.PrepareFlow(nasIP, committedFileNames)
	if err != nil {
		return
	}
	m.Stop(fmt.Sprintf("%s подготовка flow", nasIP))
	u.log.WithFields(lf).Debugf("размер flow %d", len([]rune(flow)))

	// весь tmp состоит из уже закоммиченных файлов: предыдущий цикл упал
	// между коммитом чанков и очисткой tmp. Считать нечего, нужно лишь завершить очистку
	if !hasNewFile(fileNameList, committedFileNames) {
		u.log.WithFields(lf).Debugln("новых flow файлов нет, повторная очистка tmp")
		u.removeOldFlow(nasIP)

		return
	}

	parseFlowLogName := fmt.Sprintf("%s парсинг flow, подсчет трафика", nasIP)
	m.Start(parseFlowLogName)
	trafficMap, err := u.Bridge.Traffic.ParseFlow(channelMap, flow)
	if err != nil {
		return
	}
	m.Stop(parseFlowLogName)
	u.log.WithFields(lf).Debugf("количество трафика %d", len(trafficMap))

	siftTrafficLogName := fmt.Sprintf("%s привязка трафика к сессии", nasIP)
	m.Start(siftTrafficLogName)
	chunkList, err := u.Bridge.Traffic.SiftTraffic(channelMap, trafficMap, sessionList)
	if err != nil {
		return
	}
	m.Stop(siftTrafficLogName)
	u.log.WithFields(lf).Debugf("количество чанков %d", len(chunkList))
	u.log.Debugln("актуальный результат", dump.Struct(chunkList))

	saveChunkListLogName := fmt.Sprintf("%s сохранение чанков сессии в бд", nasIP)
	m.Start(saveChunkListLogName)
	if err = u.saveChunkListWithCheckpoint(nasIP, chunkList, fileNameList); err != nil {
		return
	}
	m.Stop(saveChunkListLogName)

	u.removeOldFlow(nasIP)

	m.Result()
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
		"nas_ip": nasIP,
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
		"nas_ip": nasIP,
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
		"nas_ip": nasIP,
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
