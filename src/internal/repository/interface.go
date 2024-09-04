package repository

import (
	"aggregator/src/internal/entity/channel"
	"aggregator/src/internal/entity/session"

	"aggregator/src/internal/transaction"
)

type Channel interface {
	LoadChannelList(ts transaction.Session) (channelList []channel.Channel, err error)
}

type Session interface {
	LoadOnlineSessionList(ts transaction.Session) (sessList []session.OnlineSession, err error)
	SaveChunkList(ts transaction.Session, chunkList []session.Chunk) error
}

type Flow interface {
	ReadFlowDirNames() (dirNameList []string, err error)
	ReadFileNamesInFlowDir(dirName string) (fileNameList []string, err error)
	MoveFlowToTempDir(dirName, fileName string) error
	ReadFlow(dirName string) (output string, err error)
}
