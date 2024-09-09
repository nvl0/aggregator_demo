package usecase

import (
	"aggregator/src/bimport"
	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/transaction"
	"aggregator/src/rimport"
	"aggregator/src/tools/measure"
	"context"
	"fmt"
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

// Start запуск агрегатора
func (u *AggregatorUsecase) Start(ctx context.Context) {
	var wg sync.WaitGroup
	chanChan := make(chan map[channel.ChannelID]bool)
	sessChan := make(chan map[string][]session.OnlineSession)

	wg.Add(2)

	// получение мапки каналов
	go u.loadChannelMap(&wg, chanChan)
	// получение мапки сессий
	go u.loadOnlineSessionMap(&wg, sessChan)

	// остановка loop
	go func() {
		wg.Wait()
		close(chanChan)
		close(sessChan)
	}()

	u.measure.Start("получение списка директорий")
	dirList, err := u.Repository.Flow.ReadFlowDirNames()
	if err != nil {
		u.log.Debugln("не удалось загрузить список nas_ip директорий, ошибка", err)
		return
	}
	u.measure.Stop("получение списка директорий")

	u.log.Debugf("количество директорий %d", len(dirList))

	var (
		channelMap                     map[channel.ChannelID]bool
		sessionMap                     map[string][]session.OnlineSession
		chanChanClosed, sessChanClosed bool
	)

loop:
	for {
		select {
		case channelMap, chanChanClosed = <-chanChan:
			if channelMap == nil {
				return
			}
		case sessionMap, sessChanClosed = <-sessChan:
			if sessionMap == nil {
				return
			}
		default:
			if chanChanClosed && sessChanClosed {
				break loop
			}
		}
	}

	u.measure.Result()

	// название директорий совпадает с session.NasIP
	// если директория не совпадет с session.NasIP
	// то обработка будет отброшена
	wg.Add(len(dirList))

	for _, nasIP := range dirList {
		sessionList, exists := sessionMap[nasIP]
		if !exists {
			u.log.WithField("nas_ip", nasIP).Debugf("nas_ip %s отсутствует в бд", nasIP)
			continue
		}

		go u.Aggregate(u.SessionManager.CreateSession(), &wg, nasIP, sessionList, channelMap)
	}

	wg.Wait()
}

// loadChannelMap загрузка каналов
func (u *AggregatorUsecase) loadChannelMap(wg *sync.WaitGroup,
	chanChan chan<- map[channel.ChannelID]bool) {
	defer wg.Done()

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
		chanChan <- nil
		return
	}

	chanChan <- channelMap
}

// loadOnlineSessionMap загрузка онлайн сессий
func (u *AggregatorUsecase) loadOnlineSessionMap(wg *sync.WaitGroup,
	sessChan chan<- map[string][]session.OnlineSession) {
	defer wg.Done()

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
		sessChan <- nil
		return
	}

	sessChan <- sessionMap
}

// Aggregate агрегация траффика
func (u *AggregatorUsecase) Aggregate(ts transaction.Session, wg *sync.WaitGroup,
	nasIP string, sessionList []session.OnlineSession, channelMap map[channel.ChannelID]bool) {
	defer wg.Done()

	writer := measure.NewLogrusWriter(u.log)
	m := measure.NewMeasure(writer)

	lf := logrus.Fields{
		"nas_ip": nasIP,
	}

	u.log.WithFields(lf).Debugf("количество сессий онлайн %d", len(sessionList))

	m.Start(fmt.Sprintf("%s подготовка flow", nasIP))
	flow, err := u.Bridge.Flow.PrepareFlow(nasIP)
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

	m.Result()
}
