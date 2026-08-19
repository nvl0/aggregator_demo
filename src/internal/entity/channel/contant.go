package channel

// ID id канала
// определяющего направление трафика
type ID int

const (
	Internal ID = iota + 1
	External
)
