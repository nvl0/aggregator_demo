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

// FlowBatch чекпоинт flow файлов, чанки которых уже закоммичены
type FlowBatch interface {
	// LoadCommittedFileNames имена закоммиченных flow файлов по nas_ip.
	// Отсутствие записей — не ошибка, возвращается пустая мапка
	LoadCommittedFileNames(ts transaction.Session, nasIP string) (fileNameSet map[string]bool, err error)
	// SaveFileNames запись имен flow файлов, повторная запись известного имени не ошибка
	SaveFileNames(ts transaction.Session, nasIP string, fileNameList []string) error
	// RemoveByNasIP удаление всех записей чекпоинта по nas_ip
	RemoveByNasIP(ts transaction.Session, nasIP string) error
}

type Flow interface {
	ReadFlowDirNames() (dirNameList []string, err error)
	ReadFileNamesInFlowDir(dirName string) (fileNameList []string, err error)
	MoveFlowToTempDir(dirName, fileName string) error
	ReadFlow(dirName string, skipFileNames map[string]bool) (output string, fileNameList []string, err error)
	RemoveOld(nasIP string) (err error)
}
