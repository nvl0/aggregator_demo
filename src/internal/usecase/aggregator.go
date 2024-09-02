package usecase

import (
	"aggregator/src/bimport"
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/transaction"
	"aggregator/src/rimport"
	"aggregator/src/tools/measure"
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

func (u *AggregatorUsecase) logPrefix() string {
	return "[aggregator_usecase]"
}

// Start запуск агрегатора
func (u *AggregatorUsecase) Start() {
	ts := u.SessionManager.CreateSession()

	var (
		wg       sync.WaitGroup
		sessChan = make(chan map[string][]session.Session, 1)
	)

	wg.Add(1)

	go u.loadOnlineSessionListByNasIP(ts, &wg, sessChan)

	u.measure.Start("получение списка директорий")
	dirNasIpList, err := u.Repository.Flow.ReadFlowDirNames()
	if err != nil {
		u.log.Debugln(
			u.logPrefix(),
			fmt.Sprintf("не удалось загрузить список nas_ip директорий; ошибка: %v", err),
		)
		return
	}
	u.measure.Stop("получение списка директорий")

	u.log.Debugln(u.logPrefix(), fmt.Sprintf("количество директорий %d", len(dirNasIpList)))

	wg.Wait()
	sessionMap := <-sessChan
	if sessionMap == nil {
		return
	}

	u.measure.Result()

	wg.Add(len(dirNasIpList))

	// название директорий совпадает с session.NasIP
	// если директория не совпадет с session.NasIP
	// то обработка будет отброшена
	for _, nasIP := range dirNasIpList {
		go u.aggregate(ts.CreateNewSession(), &wg, nasIP, sessionMap)
	}

	wg.Wait()
}

// loadOnlineSessionListByNasIP загрузка онлайн сессий
func (u *AggregatorUsecase) loadOnlineSessionListByNasIP(ts transaction.Session, wg *sync.WaitGroup,
	sessChan chan<- map[string][]session.Session) {

	defer wg.Done()

	if err := ts.Start(); err != nil {
		u.log.Errorln(
			u.logPrefix(),
			fmt.Sprintf("не удалось открыть транзакцию; ошибка: %v", err),
		)
		return
	}
	defer ts.Rollback()

	u.measure.Start("получение списка онлайн сессий")
	defer u.measure.Stop("получение списка онлайн сессий")

	sessionMap, err := u.Bridge.Session.LoadOnlineSessionListByNasIP(ts)
	if err != nil {
		u.log.Errorln(
			u.logPrefix(),
			fmt.Sprintf("не удалось загрузить список онлайн сессий; ошибка: %v", err),
		)

		sessChan <- nil
		return
	}

	sessChan <- sessionMap
}

// aggregate агрегация трафика
func (u *AggregatorUsecase) aggregate(ts transaction.Session, wg *sync.WaitGroup, nasIP string, sessionMap map[string][]session.Session) {
	defer wg.Done()

	writer := measure.NewLogrusWriter(u.log)
	m := measure.NewMeasure(writer)

	lf := logrus.Fields{
		"nas_ip": nasIP,
	}

	sessionList, exists := sessionMap[nasIP]
	if !exists {
		u.log.WithFields(lf).Debugln(
			u.logPrefix(), fmt.Sprintf("nas_ip %s отсутствует в бд", nasIP),
		)
		return
	}
	u.log.Debugln(u.logPrefix(), fmt.Sprintf("nas_ip %s; количество сессий онлайн %d", nasIP, len(sessionList)))

	m.Start(fmt.Sprintf("%s подготовка flow", nasIP))
	flow, err := u.Bridge.Flow.PrepareFlow(nasIP)
	if err != nil {
		return
	}
	m.Stop(fmt.Sprintf("%s подготовка flow", nasIP))
	u.log.Debugln(u.logPrefix(), fmt.Sprintf("nas_ip %s; размер flow %d", nasIP, len([]rune(flow))))

	m.Start(fmt.Sprintf("%s парсинг flow, подсчет трафика", nasIP))
	trafficMap, err := u.Bridge.Traffic.ParseFlow(flow)
	if err != nil {
		return
	}
	m.Stop(fmt.Sprintf("%s парсинг flow, подсчет трафика", nasIP))
	u.log.Debugln(u.logPrefix(), fmt.Sprintf("nas_ip %s; количество трафика %d", nasIP, len(trafficMap)))

	m.Start(fmt.Sprintf("%s привязка трафика к сессии", nasIP))
	chunkList, err := u.Bridge.Traffic.SiftTraffic(trafficMap, sessionList)
	if err != nil {
		return
	}
	m.Stop(fmt.Sprintf("%s привязка трафика к сессии", nasIP))
	u.log.Debugln(u.logPrefix(), fmt.Sprintf("nas_ip %s; количество чанков %d", nasIP, len(chunkList)))

	if err = ts.Start(); err != nil {
		u.log.Errorln(
			u.logPrefix(),
			fmt.Sprintf("не удалось открыть транзакцию; ошибка: %v", err),
		)
		return
	}
	defer ts.Rollback()

	m.Start(fmt.Sprintf("%s сохранение чанков сессии в бд", nasIP))
	if err = u.Repository.Session.SaveChunkList(ts, chunkList); err != nil {
		u.log.WithFields(lf).Errorln(
			u.logPrefix(),
			fmt.Sprintf("не удалось сохранить чанки; ошибка: %v", err),
		)
		return
	}
	m.Stop(fmt.Sprintf("%s сохранение чанков сессии в бд", nasIP))

	if err = ts.Commit(); err != nil {
		u.log.Errorln(
			u.logPrefix(),
			fmt.Sprintf("не удалось закрыть транзакцию; ошибка: %v", err),
		)
		return
	}

	m.Result()
}
