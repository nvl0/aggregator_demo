package channel

type Channel struct {
	ID      ChannelID `db:"channel_id"`
	Enabled bool      `db:"enabled"`
	Descr   string    `db:"descr"`
}
