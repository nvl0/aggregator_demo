package repository

import (
	"aggregator/app/internal/entity/session"

	"aggregator/app/internal/transaction"
)

type Session interface {
	LoadOnlineSessionList(ts transaction.Session) (sessList []session.Session, err error)
	SaveChunkList(ts transaction.Session, chunkList []session.Chunk) error
}

type Flow interface {
	ReadFlowDirNames() (dirNameList []string, err error)
	ReadFileNamesInFlowDir(dirName string) (fileNameList []string, err error)
	MoveFlowToTempDir(dirName, fileName string) error
	ReadFlow(dirName string) (output string, err error)
}
