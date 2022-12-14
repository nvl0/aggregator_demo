package uimport

import "aggregator/app/internal/usecase"

type Usecase struct {
	Flow       *usecase.FlowUsecase
	Session    *usecase.SessionUsecase
	Traffic    *usecase.TrafficUsecase
	Aggregator *usecase.AggregatorUsecase
}
