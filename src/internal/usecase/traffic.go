package usecase

import (
	"log/slog"
	"net"
	"strconv"
	"strings"

	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/flow"
	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/entity/traffic"
	"aggregator/src/rimport"

	"github.com/yl2chen/cidranger"
)

// flowRowFieldCount количество полей в строке flow: #:doctets,srcaddr,dstaddr
const flowRowFieldCount = 3

type TrafficUsecase struct {
	log *slog.Logger
	rimport.RepositoryImports
	internalNet cidranger.Ranger
}

func NewTrafficUsecase(
	log *slog.Logger,
	ri rimport.RepositoryImports,
	internalNet cidranger.Ranger,
) *TrafficUsecase {
	return &TrafficUsecase{
		log:               log,
		RepositoryImports: ri,
		internalNet:       internalNet,
	}
}

// FlowAccumulator построчный аккумулятор трафика одного nas_ip.
// Состояние живет здесь, а не в TrafficUsecase: usecase один на все воркеры пула,
// а nas_ip обрабатываются параллельно
type FlowAccumulator struct {
	u          *TrafficUsecase
	channelMap map[channel.ID]bool
	// trafficMap map[user_ip]map[channel_id]Traffic
	trafficMap map[session.IP]map[channel.ID]traffic.Traffic
	// parsedRecords количество успешно распарсенных строк
	parsedRecords int
	// classifyErrCount количество строк, которые не удалось классифицировать по блоку internal
	classifyErrCount int
}

// NewFlowAccumulator аккумулятор трафика на один nas_ip
func (u *TrafficUsecase) NewFlowAccumulator(channelMap map[channel.ID]bool) *FlowAccumulator {
	return &FlowAccumulator{
		u:          u,
		channelMap: channelMap,
		trafficMap: make(map[session.IP]map[channel.ID]traffic.Traffic),
	}
}

// AccumulateLine разбор одной строки flow и учет ее в трафике.
// Некорректная строка не является ошибкой: она пропускается и только считается,
// как и в теле цикла прежнего ParseFlow
func (a *FlowAccumulator) AccumulateLine(line string) error {
	// flow собирается с нескольких файлов
	// в каждом файле есть заголовок
	// #:doctets,srcaddr,dstaddr
	if strings.Contains(line, flow.FlowHeader) {
		return nil
	}

	// ряд который содержит \t или \n не будет считан
	if line == "" {
		return nil
	}

	// определение аргументов в ряду
	rowArgs := strings.Split(line, ",")
	if len(rowArgs) != flowRowFieldCount {
		return nil
	}

	var bytes, srcIP, dstIP = rowArgs[0], rowArgs[1], rowArgs[2]

	// парсинг аргументов
	record, err := a.u.parseRecord(bytes, srcIP, dstIP)
	if err != nil {
		a.u.log.Warn("обнаружена некорректная запись flow, ошибка",
			"error", err, "bytes", bytes, "src_ip", srcIP, "dst_ip", dstIP)
		return nil
	}

	a.parsedRecords++

	// определение принадлежности отправителя/получателя к сети.
	// ошибку не логируем построчно (систематический сбой затопит лог) —
	// копим счетчик и пишем один итог в Result
	isSrcInternal, containsErr := a.u.internalNet.Contains(record.SrcIP)

	var isDstInternal bool
	if containsErr == nil {
		isDstInternal, containsErr = a.u.internalNet.Contains(record.DstIP)
	}

	if containsErr != nil {
		a.classifyErrCount++
		return nil //nolint:nilerr // ошибка классификации сети не прерывает разбор, копится в classifyErrCount
	}

	a.count(record, isSrcInternal, isDstInternal)

	return nil
}

// count запись трафика записи по направлениям
func (a *FlowAccumulator) count(record flow.Record, isSrcInternal, isDstInternal bool) {
	switch {
	// получатель и отправитель внутри сети internal
	case isSrcInternal && isDstInternal:
		// запись получателю в download
		a.trafficMap[record.SrcIPkey()] = a.u.CountTraffic(
			a.trafficMap[record.SrcIPkey()],
			traffic.NewTrafficDownload(record.ByteSize),
			a.channelMap,
			channel.Internal,
		)

		// запись отправителю в upload
		a.trafficMap[record.DstIPkey()] = a.u.CountTraffic(
			a.trafficMap[record.DstIPkey()],
			traffic.NewTrafficUpload(record.ByteSize),
			a.channelMap,
			channel.Internal,
		)

	// получатель внутри сети internal
	case isSrcInternal:
		// отправитель во внешней сети
		a.trafficMap[record.SrcIPkey()] = a.u.CountTraffic(
			a.trafficMap[record.SrcIPkey()],
			traffic.NewTrafficDownload(record.ByteSize),
			a.channelMap,
			channel.External,
		)

	// отправитель внутри сети internal
	case isDstInternal:
		// получатель во внешней сети
		a.trafficMap[record.DstIPkey()] = a.u.CountTraffic(
			a.trafficMap[record.DstIPkey()],
			traffic.NewTrafficUpload(record.ByteSize),
			a.channelMap,
			channel.External,
		)
	}
}

// Result итог разбора: накопленный трафик и статус.
// global.ErrNoData — строки распарсились, но internal трафика нет (flow только с external);
// global.ErrInternalError — ни одной валидной строки либо ни одну не удалось классифицировать
func (a *FlowAccumulator) Result() (
	trafficMap map[session.IP]map[channel.ID]traffic.Traffic, err error) {
	if a.classifyErrCount > 0 {
		a.u.log.Warn("не удалось проверить принадлежность IP к блоку internal, строки пропущены",
			"count", a.classifyErrCount)
	}

	if len(a.trafficMap) == 0 {
		switch {
		case a.parsedRecords == 0, a.classifyErrCount == a.parsedRecords:
			err = global.ErrInternalError
		default:
			err = global.ErrNoData
		}
	}

	return a.trafficMap, err
}

// parseRecord парсинг одной записи flow
func (u *TrafficUsecase) parseRecord(byteSizeRaw, srcIPRaw, dstIPRaw string) (r flow.Record, err error) {
	// парсинг получателя
	if r.SrcIP = net.ParseIP(srcIPRaw); r.SrcIP == nil {
		r.Empty()
		err = flow.ErrUndefinedIPFormat
		return r, err
	}

	// парсинг отправителя
	if r.DstIP = net.ParseIP(dstIPRaw); r.DstIP == nil {
		r.Empty()
		err = flow.ErrUndefinedIPFormat
		return r, err
	}

	// количество использованных байт
	if r.ByteSize, err = strconv.Atoi(byteSizeRaw); err != nil {
		r.Empty()
		err = flow.ErrTrafficByteParse
	}

	return r, err
}

// CountTraffic подсчет трафика по направлениям
func (u *TrafficUsecase) CountTraffic(oldTraffic map[channel.ID]traffic.Traffic,
	newTraffic traffic.Traffic, channelMap map[channel.ID]bool,
	channelID channel.ID) map[channel.ID]traffic.Traffic {
	// если старый трафик существует, то объединить
	if len(oldTraffic) != 0 {
		// если подсчет по каналу разрешен
		if channelMap[channelID] {
			newTraffic.Merge(oldTraffic[channelID])
			oldTraffic[channelID] = newTraffic
		}

		return oldTraffic
	}

	// если старого трафика не существует, то создать
	// новый пустой трафик по всем направлениям
	newChannelMap := u.createNewEmptyTrafficMap(channelMap)

	// однако, записан будет только newTraffic по своему напрвлению
	// если подсчет по каналу разрешен
	if channelMap[channelID] {
		newChannelMap[channelID] = newTraffic
	}

	return newChannelMap
}

// createNewEmptyTrafficMap создание пустого трафика по всем доступным направлениям
func (u *TrafficUsecase) createNewEmptyTrafficMap(channelMap map[channel.ID]bool,
) map[channel.ID]traffic.Traffic {
	trafficMap := make(map[channel.ID]traffic.Traffic, len(channelMap))

	for channelID, enabled := range channelMap {
		if enabled {
			trafficMap[channelID] = traffic.NewEmptyTraffic()
		}
	}

	return trafficMap
}

// SiftTraffic просеивание трафика для получение чанков
func (u *TrafficUsecase) SiftTraffic(channelMap map[channel.ID]bool,
	trafficMap map[session.IP]map[channel.ID]traffic.Traffic,
	sessionList []session.OnlineSession) (chunkList []session.Chunk, err error) {
	chunkList = make([]session.Chunk, 0, len(sessionList))

	for _, sess := range sessionList {
		// сессии у которых есть трафик будут записаны в чанки
		channelList, exists := trafficMap[sess.IP]

		if !exists {
			// если трафика нет, то будут заполнены нулевые значения
			// чтобы подделать активность сессии
			channelList = u.createNewEmptyTrafficMap(channelMap)
		}

		for channelID, traffic := range channelList {
			chunkList = append(chunkList,
				session.NewChunk(
					sess.SessID, int(channelID),
					traffic.Download, traffic.Upload,
				))
		}
	}

	if len(chunkList) == 0 {
		err = global.ErrNoData
		u.log.Error("не удалось просеять трафик, ошибка", "error", err, logFieldNasIP, sessionList[0].NasIP)
	}

	return chunkList, err
}
