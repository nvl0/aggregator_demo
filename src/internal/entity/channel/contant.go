package channel

// ChannelID id канала
// определяющего направление трафика
type ChannelID int

const (
	Internal ChannelID = iota + 1
	External
)
