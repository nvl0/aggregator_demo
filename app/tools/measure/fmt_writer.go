package measure

import "fmt"

type fmtWriter struct {
}

// NewFmtWriter писать в логи fmt
func NewFmtWriter() Writer {
	return &fmtWriter{}
}

func (l *fmtWriter) Write(msg string) {
	fmt.Println(msg)
}
