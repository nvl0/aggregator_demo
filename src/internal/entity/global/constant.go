package global

import "time"

// ChannelID id канала
// определяющего направление трафика
type ChannelID int

// SubnetFileName название файла с подсетями
type SubnetFileName string

const (
	// InternalDisabled имя файла с исключенными из отчета подсетями
	InternalDisabled SubnetFileName = "internal"
)

const (
	// StartDur время, через которое агрегатор будет перезапущен
	StartDur = 5 * time.Second
	// ProcessTimeout время, через которое агрегатор будет завершен
	ProcessTimeout = 7 * time.Second
)
