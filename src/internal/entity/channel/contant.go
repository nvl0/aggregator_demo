package channel

// ChannelID id канала
// определяющего направление трафика
//
//nolint:revive // переименование в ID потребовало бы правки всех 12 файлов, использующих channel.ChannelID как публичный тип
type ChannelID int

const (
	Internal ChannelID = iota + 1
	External
)
