package channel

type Channel struct {
	ID      ID     `db:"channel_id"`
	Enabled bool   `db:"enabled"`
	Descr   string `db:"descr"`
}
