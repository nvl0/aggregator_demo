package global

import "errors"

var (
	// ErrDBUnvailable база данных недоступна
	ErrDBUnvailable = errors.New("база данных недоступна")

	// ErrInternalError внутряя ошибка
	ErrInternalError = errors.New("произошла внутреняя ошибка, пожалуйста попробуйте выполнить действие позже")

	// ErrNoData данные не найдены
	ErrNoData = errors.New("данные не найдены")
)
