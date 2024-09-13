package usecase

import (
	"aggregator/src/bimport"
	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/flow"
	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/entity/traffic"
	"aggregator/src/rimport"
	"net"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/yl2chen/cidranger"
)

type TrafficUsecase struct {
	log *logrus.Logger
	rimport.RepositoryImports
	*bimport.BridgeImports
	internalNet cidranger.Ranger
}

func NewTrafficUsecase(
	log *logrus.Logger,
	ri rimport.RepositoryImports,
	bi *bimport.BridgeImports,
	internalNet cidranger.Ranger,
) *TrafficUsecase {
	return &TrafficUsecase{
		log:               log,
		RepositoryImports: ri,
		BridgeImports:     bi,
		internalNet:       internalNet,
	}
}

// ParseFlow парсинг flow
// trafficMap map[user_ip]map[channel_id]Traffic
func (u *TrafficUsecase) ParseFlow(channelMap map[channel.ChannelID]bool, flowStr string) (
	trafficMap map[session.IP]map[channel.ChannelID]traffic.Traffic, err error) {
	var (
		// обозначение принадлежности получателя/отправителя к сети
		isSrcInternal, isDstInternal bool
		// строчный ряд при считывании flow
		row string
		// аргументы в ряду слева направо
		rowArgs []string
		// запись полученная при парсинге агрументов одного ряда
		record flow.Record
	)

	// построчная разбивка flowStr
	// flowStr представляет собой таблицу
	flowArr := strings.Split(flowStr, "\n")

	// при парсинге flow первая строка состоит
	// из заголовка #:doctets,srcaddr,dstaddr
	if strings.Contains(string(flowArr[0]), flow.FlowHeader) {
		flowArr = flowArr[1:]
	}

	trafficMap = make(map[session.IP]map[channel.ChannelID]traffic.Traffic, len(flowArr))

	// парсинг flow
	for _, row = range flowArr {
		// ряд который содержит \t или \n не будет считан
		if row != "" {
			// определение аргументов в ряду
			if rowArgs = strings.Split(row, ","); len(rowArgs) == 3 {
				var bytes, srcIP, dstIP = rowArgs[0], rowArgs[1], rowArgs[2]

				lf := logrus.Fields{
					"bytes":  bytes,
					"src_ip": srcIP,
					"dst_ip": dstIP,
				}

				// парсинг аргументов
				if record, err = u.parseRecord(bytes, srcIP, dstIP); err != nil {
					u.log.WithFields(lf).Warnln(flow.ErrIncorrectRecord(err))
					continue
				}

				// определение принадлежности отправителя/получателя к сети
				isSrcInternal, _ = u.internalNet.Contains(record.SrcIP)
				isDstInternal, _ = u.internalNet.Contains(record.DstIP)

				switch {
				// получатель и отправитель внутри сети internal
				case isSrcInternal && isDstInternal:

					// запись получателю в download
					trafficMap[session.IP(record.SrcIPkey())] = u.Bridge.Traffic.CountTraffic(
						trafficMap[session.IP(record.SrcIPkey())],
						traffic.NewTrafficDownload(record.ByteSize),
						channelMap,
						channel.Internal,
					)

					// запись отправителю в upload
					trafficMap[session.IP(record.DstIPkey())] = u.Bridge.Traffic.CountTraffic(
						trafficMap[session.IP(record.DstIPkey())],
						traffic.NewTrafficUpload(record.ByteSize),
						channelMap,
						channel.Internal,
					)

				// получатель внутри сети internal
				case isSrcInternal:

					// отправитель во внешней сети
					trafficMap[session.IP(record.SrcIPkey())] = u.Bridge.Traffic.CountTraffic(
						trafficMap[session.IP(record.SrcIPkey())],
						traffic.NewTrafficDownload(record.ByteSize),
						channelMap,
						channel.External,
					)

				// отправитель внутри сети internal
				case isDstInternal:

					// получатель во внешней сети
					trafficMap[session.IP(record.DstIPkey())] = u.Bridge.Traffic.CountTraffic(
						trafficMap[session.IP(record.DstIPkey())],
						traffic.NewTrafficUpload(record.ByteSize),
						channelMap,
						channel.External,
					)
				}
			}
		}
	}

	if len(trafficMap) == 0 {
		err = global.ErrNoData
	}

	return
}

// parseRecord парсинг одной записи flow
func (u *TrafficUsecase) parseRecord(byteSizeRaw, srcIpRaw, dstIpRaw string) (r flow.Record, err error) {
	// парсинг получателя
	if r.SrcIP = net.ParseIP(srcIpRaw); r.SrcIP == nil {
		r.Empty()
		err = flow.ErrUndefinedIpFormat
		return
	}

	// парсинг отправителя
	if r.DstIP = net.ParseIP(dstIpRaw); r.DstIP == nil {
		r.Empty()
		err = flow.ErrUndefinedIpFormat
		return
	}

	// количество использованных байт
	if r.ByteSize, err = strconv.Atoi(byteSizeRaw); err != nil {
		r.Empty()
		err = flow.ErrTrafficByteParse
	}

	return
}

// CountTraffic подсчет трафика по направлениям
func (u *TrafficUsecase) CountTraffic(oldTraffic map[channel.ChannelID]traffic.Traffic,
	newTraffic traffic.Traffic, channelMap map[channel.ChannelID]bool,
	channelID channel.ChannelID) map[channel.ChannelID]traffic.Traffic {

	// если старый трафик существует, то объединить
	if len(oldTraffic) != 0 {

		// если подсчет по каналу разрешен
		if channelMap[channelID] {
			newTraffic.Merge(oldTraffic[channelID])
			oldTraffic[channelID] = newTraffic
		}

		return oldTraffic

	} else {
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
}

// createNewEmptyTrafficMap создание пустого трафика по всем доступным направлениям
func (u *TrafficUsecase) createNewEmptyTrafficMap(channelMap map[channel.ChannelID]bool,
) map[channel.ChannelID]traffic.Traffic {
	trafficMap := make(map[channel.ChannelID]traffic.Traffic, len(channelMap))

	for channelID, enabled := range channelMap {
		if enabled {
			trafficMap[channelID] = traffic.NewEmptyTraffic()
		}
	}

	return trafficMap
}

// SiftTraffic просеивание трафика для получение чанков
func (u *TrafficUsecase) SiftTraffic(channelMap map[channel.ChannelID]bool,
	trafficMap map[session.IP]map[channel.ChannelID]traffic.Traffic,
	sessionList []session.OnlineSession) (chunkList []session.Chunk, err error) {

	lf := logrus.Fields{
		"nas_ip": sessionList[0].NasIP,
	}

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
		u.log.WithFields(lf).Errorln("не удалось просеять трафик, ошибка", err)
	}

	return
}
