package channel

type Channel struct {
	ID      int    `db:"channel_id"`
	Enabled bool   `db:"enabled"`
	Descr   string `db:"descr"`
}
