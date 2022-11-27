package external

import (
	"aggregator/app/internal/entity/global"
	"aggregator/app/uimport"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Cron struct {
	log *logrus.Logger
	uimport.UsecaseImports
	mu *sync.Mutex
}

func NewCron(log *logrus.Logger,
	u uimport.UsecaseImports) *Cron {
	return &Cron{
		log:            log,
		UsecaseImports: u,
		mu:             &sync.Mutex{},
	}
}

func (c *Cron) Run() {
	for range time.Tick(time.Second * global.AggregatorStartSeconds) {
		c.mu.Lock()
		c.Usecase.Aggregator.Start()
		c.mu.Unlock()
	}
}
