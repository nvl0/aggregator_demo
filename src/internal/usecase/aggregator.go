package usecase

import (
	"aggregator/src/bimport"
	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/session"
	"aggregator/src/rimport"
	"aggregator/src/tools/dump"
	"aggregator/src/tools/flowgen"
	"aggregator/src/tools/measure"
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

type AggregatorUsecase struct {
	measure measure.Measure
	log     *logrus.Logger
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
		RepositoryImports: ri,
		BridgeImports:     bi,
	}
}

var fgen = os.Getenv("FLOWGEN") == "true"

// Start запуск агрегатора
func (u *AggregatorUsecase) Start(ctx context.Context) {
	// генерация flow
	if fgen {
		if down, up, err := flowgen.Generate(); err == nil {
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

	chanChan := make(chan map[channel.ChannelID]bool)
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

	var wg sync.WaitGroup
	done := make(chan struct{})

	// название директории совпадает с session.NasIP
	for _, nasIP := range dirList {

		// если директория не совпадет с session.NasIP
		// то обработка директории будет отброшена
		sessionList, exists := sessionMap[nasIP]
		if !exists {
			u.log.WithField("nas_ip", nasIP).Debugf("nas_ip %s отсутствует в бд", nasIP)
			continue
		}

		wg.Add(1)
		go u.Bridge.Aggregator.Aggregate(&wg, nasIP, sessionList, channelMap)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

loop:
	for {
		select {
		case <-done:
			break loop
		case <-ctx.Done():
			break loop
		}
	}
}

// loadChannelMap загрузка каналов
func (u *AggregatorUsecase) loadChannelMap(chanChan chan<- map[channel.ChannelID]bool) {
	defer close(chanChan)

	ts := u.SessionManager.CreateSession()
	if err := ts.Start(); err != nil {
		u.log.Errorln("не удалось открыть транзакцию, ошибка", err)
		return
	}
	defer ts.Rollback()

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
	defer ts.Rollback()

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
func (u *AggregatorUsecase) Aggregate(wg *sync.WaitGroup, nasIP string, sessionList []session.OnlineSession, channelMap map[channel.ChannelID]bool) {
	defer wg.Done()

	writer := measure.NewLogrusWriter(u.log)
	m := measure.NewMeasure(writer)

	lf := logrus.Fields{
		"nas_ip": nasIP,
	}

	u.log.WithFields(lf).Debugf("количество сессий онлайн %d", len(sessionList))

	m.Start(fmt.Sprintf("%s подготовка flow", nasIP))
	flow, err := u.Bridge.Flow.PrepareFlow(string(nasIP))
	if err != nil {
		return
	}
	m.Stop(fmt.Sprintf("%s подготовка flow", nasIP))
	u.log.WithFields(lf).Debugf("размер flow %d", len([]rune(flow)))

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

	ts := u.SessionManager.CreateSession()
	if err = ts.Start(); err != nil {
		u.log.Errorln("не удалось открыть транзакцию, ошибка", err)
		return
	}
	defer ts.Rollback()

	saveChunkListLogName := fmt.Sprintf("%s сохранение чанков сессии в бд", nasIP)
	m.Start(saveChunkListLogName)
	if err = u.Repository.Session.SaveChunkList(ts, chunkList); err != nil {
		u.log.WithFields(lf).Errorln("не удалось сохранить чанки, ошибка", err)
		return
	}
	m.Stop(saveChunkListLogName)

	if err = ts.Commit(); err != nil {
		u.log.Errorln("не удалось закрыть транзакцию, ошибка", err)
		return
	}

	if err = u.Repository.Flow.RemoveOld(nasIP); err != nil {
		u.log.WithFields(lf).Errorln("не удалось удалить старый flow, ошибка", err)
		err = nil
	}

	m.Result()
}
