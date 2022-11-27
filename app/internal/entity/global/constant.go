package global

// ChannelID канал определяющий направление трафика
type ChannelID int

const (
	// Internal ChannelID id внутреннего канала
	Internal ChannelID = 0
	// Internet ChannelID id внешнего канала
	Internet ChannelID = 1
)

// AllChannelIDList список всех каналов
var AllChannelIDList = []ChannelID{Internet, Internal}

// EnabledChannelIDMap контроль активных каналов
var EnabledChannelIDMap = map[ChannelID]bool{
	Internet: true,
	Internal: false,
}

// SubnetFileName название файла с подсетями
type SubnetFileName string

const (
	// InternalDisabled имя файла с исключенными из отчета подсетями
	InternalDisabled SubnetFileName = "internal"
)

// AggregatorStartSeconds время через которое агрегатор будет перезапущен
const AggregatorStartSeconds = 180
