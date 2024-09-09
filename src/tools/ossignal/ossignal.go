package ossignal

import (
	"os"
	"os/signal"
	"syscall"
)

func WaitForTerm(flag chan<- struct{}) {
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, syscall.SIGTERM)
	<-s
	close(flag)
}
