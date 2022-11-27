package flow

import (
	"errors"
	"fmt"
)

var (
	// ErrUndefinedIpFormat неизвестный формат ip-адреса
	ErrUndefinedIpFormat = errors.New("неизвестный формат ip-адреса")
	// ErrTrafficByteParse некорректный формат трафика
	ErrTrafficByteParse = errors.New("некорректный формат трафика")
	// ErrIncorrectRecord обнаружена некоректная запись flow
	ErrIncorrectRecord = func(err error) error {
		return fmt.Errorf("обнаружена некоректная запись flow, ошибка: %v", err)
	}
)
