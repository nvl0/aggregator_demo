package external

import (
	"context"
	"log/slog"
	"time"

	"aggregator/src/internal/entity/global"
	"aggregator/src/uimport"
)

type Cron struct {
	log *slog.Logger
	uimport.UsecaseImports
}

func NewCron(log *slog.Logger,
	u uimport.UsecaseImports) *Cron {
	return &Cron{
		log:            log,
		UsecaseImports: u,
	}
}

func (c *Cron) Run(ctx context.Context) {
	tick := time.NewTicker(global.StartDur)
	defer tick.Stop()

loop:
	for {
		select {
		case <-tick.C:
			c.Usecase.Aggregator.Start(ctx)
		case <-ctx.Done():
			break loop
		}
	}
}
