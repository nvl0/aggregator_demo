package transaction

import "errors"

var (
	ErrActiveTransaction = errors.New("открытие транзакции при активной транзакции")
	ErrNotInit           = errors.New("транзакция не была инициализирована")
	ErrClosed            = errors.New("транзакция была завершена")
)
