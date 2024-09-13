package session

// OnlineSession сессия клиента
type OnlineSession struct {
	SessID int   `db:"sess_id"`
	IP     IP    `db:"ip"`
	NasIP  NasIP `db:"nas_ip"`
}

// Chunk чанк с сформированным и отсеяным трафиком по направлению
type Chunk struct {
	SessID    int `db:"sess_id"`
	ChannelID int `db:"channel_id"`
	Download  int `db:"upload"`
	Upload    int `db:"download"`
}

// NewChunk конструктор
func NewChunk(sessID, channelID, download, upload int) Chunk {
	return Chunk{
		SessID:    sessID,
		ChannelID: channelID,
		Download:  download,
		Upload:    upload,
	}
}
