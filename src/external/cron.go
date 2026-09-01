package external

import (
	"context"
	"time"

	"aggregator/src/internal/entity/global"
	"aggregator/src/uimport"

	"github.com/sirupsen/logrus"
)

type Cron struct {
	log *logrus.Logger
	uimport.UsecaseImports
}

func NewCron(log *logrus.Logger,
	u uimport.UsecaseImports) *Cron {
	return &Cron{
		log:            log,
		UsecaseImports: u,
	}
}

func (c *Cron) Run(termFlag <-chan struct{}) {
	tick := time.NewTicker(global.StartDur)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

loop:
	for {
		select {
		case <-tick.C:
			c.Usecase.Aggregator.Start(ctx)
		case <-termFlag:
			break loop
		case <-ctx.Done():
			break loop
		}
	}
}
