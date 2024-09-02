package usecase

import (
	"aggregator/src/bimport"
	"aggregator/src/internal/entity/flow"
	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/entity/session"
	"aggregator/src/internal/entity/traffic"
	"aggregator/src/rimport"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/yl2chen/cidranger"
)

type TrafficUsecase struct {
	log *logrus.Logger
	//
	rimport.RepositoryImports
	*bimport.BridgeImports
	//
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

func (u *TrafficUsecase) logPrefix() string {
	return "[traffic_usecase]"
}

// ParseFlow парсинг flow
func (u *TrafficUsecase) ParseFlow(flowStr string) (trafficMap map[string]map[global.ChannelID]traffic.Traffic, err error) {
	trafficMap = make(map[string]map[global.ChannelID]traffic.Traffic)

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

	// парсинг flow
	for _, row = range strings.Split(flowStr, "\n") {
		// при парсинге flow первая строка состоит
		// из заголовка #:doctets,srcaddr,dstaddr
		if strings.Contains(row, flow.FlowHeader) {
			continue
		}

		// ряд который содержит \t или \n не будет считан
		if row != "" {

			// определение аргументов в ряду
			if rowArgs = strings.Split(row, ","); len(rowArgs) == 3 {
				lf := logrus.Fields{
					"bytes":  rowArgs[0],
					"src_ip": rowArgs[1],
					"dst_ip": rowArgs[2],
				}

				// парсинг аргументов
				if record, err = u.parseRecord(rowArgs[0], rowArgs[1], rowArgs[2]); err != nil {
					u.log.WithFields(lf).Warnln(u.logPrefix(), flow.ErrIncorrectRecord(err))
					continue
				}

				// определение принадлежности отправителя/получателя к сети
				isSrcInternal, _ = u.internalNet.Contains(record.SrcIP)
				isDstInternal, _ = u.internalNet.Contains(record.DstIP)

				switch {

				// получатель и отправитель внутри сети internal
				case isSrcInternal && isDstInternal:

					// запись получателю в download
					trafficMap[record.SrcIPkey()] = u.countTraffic(
						trafficMap[record.SrcIPkey()],
						traffic.NewTrafficDownload(record.ByteSize),
						global.Internal,
					)

					// запись отправителю в upload
					trafficMap[record.DstIPkey()] = u.countTraffic(
						trafficMap[record.DstIPkey()],
						traffic.NewTrafficUpload(record.ByteSize),
						global.Internal,
					)

				// получатель внутри сети internal
				case isSrcInternal:

					// отправитель во внешней сети
					trafficMap[record.SrcIPkey()] = u.countTraffic(
						trafficMap[record.SrcIPkey()],
						traffic.NewTrafficDownload(record.ByteSize),
						global.Internet,
					)

				// отправитель внутри сети internal
				case isDstInternal:

					// получатель во внешней сети
					trafficMap[record.DstIPkey()] = u.countTraffic(
						trafficMap[record.DstIPkey()],
						traffic.NewTrafficUpload(record.ByteSize),
						global.Internet,
					)
				}
			}
		}
	}

	if len(trafficMap) == 0 {
		err = global.ErrNoData
		u.log.Errorln(
			u.logPrefix(),
			fmt.Sprintf("не удалось посчитать трафик; ошибка: %v", err),
		)
	}

	return
}

// parseRecord парсинг одной записи flow
func (u *TrafficUsecase) parseRecord(byteSizeRaw, srcIpRaw, dstIpRaw string) (r flow.Record, err error) {
	// парсинг получателя
	if r.SrcIP = net.ParseIP(srcIpRaw); r.SrcIP == nil {
		r = flow.Record{}
		err = flow.ErrUndefinedIpFormat
		return
	}

	// парсинг отправителя
	if r.DstIP = net.ParseIP(dstIpRaw); r.DstIP == nil {
		r = flow.Record{}
		err = flow.ErrUndefinedIpFormat
		return
	}

	// количество использованных байт
	if r.ByteSize, err = strconv.Atoi(byteSizeRaw); err != nil {
		r = flow.Record{}
		err = flow.ErrTrafficByteParse
	}

	return
}

// countTraffic подсчет трафика по направлениям
func (u *TrafficUsecase) countTraffic(oldTraffic map[global.ChannelID]traffic.Traffic,
	newTraffic traffic.Traffic, channelID global.ChannelID) map[global.ChannelID]traffic.Traffic {

	// если старый трафик существует, то объединить
	if oldTraffic != nil {

		// если подсчет по каналу разрешен
		if global.EnabledChannelIDMap[channelID] {
			newTraffic.Merge(oldTraffic[channelID])
			oldTraffic[channelID] = newTraffic
		}

		return oldTraffic

	} else {

		// если старого трафика не существует, то создать
		// новый пустой трафик по всем направлениям
		newChannelMap := u.createNewEmptyChannelMap()

		// однако, записан будет только newTraffic по своему напрвлению
		// если подсчет по каналу разрешен
		if global.EnabledChannelIDMap[channelID] {
			newChannelMap[channelID] = newTraffic
		}

		return newChannelMap
	}
}

// createNewEmptyChannelMap создание пустого трафика по всем направлениям
func (u *TrafficUsecase) createNewEmptyChannelMap() map[global.ChannelID]traffic.Traffic {
	channelMap := make(map[global.ChannelID]traffic.Traffic)

	for _, channelID := range global.AllChannelIDList {
		if global.EnabledChannelIDMap[channelID] {
			channelMap[channelID] = traffic.NewEmptyTraffic()
		}
	}

	return channelMap
}

// SiftTraffic просеивание трафика для получение чанков
func (u *TrafficUsecase) SiftTraffic(trafficMap map[string]map[global.ChannelID]traffic.Traffic,
	sessionList []session.Session) (chunkList []session.Chunk, err error) {

	lf := logrus.Fields{
		"nas_ip": sessionList[0].NasIP,
	}

	chunkList = make([]session.Chunk, 0)

	for _, sess := range sessionList {
		// сессии у которых есть трафик будут записаны в чанки
		channelList, exists := trafficMap[sess.IP]
		if !exists {
			// если трафика нет, то будут заполнены нулевые значения
			// чтобы подделать активность сессии
			channelList = u.createNewEmptyChannelMap()
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
		u.log.WithFields(lf).Errorln(
			u.logPrefix(),
			fmt.Sprintf("не удалось просеять трафик; ошибка: %v", err),
		)
	}

	return
}
