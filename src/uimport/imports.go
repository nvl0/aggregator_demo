package uimport

import (
	"fmt"
	"log/slog"
	"os"

	"aggregator/src/config"
	"aggregator/src/internal/entity/flow"
	"aggregator/src/internal/transaction"
	"aggregator/src/internal/usecase"
	"aggregator/src/rimport"
	"aggregator/src/tools/logger"
	"aggregator/src/tools/metrics"
	"aggregator/src/tools/subnetrange"
	"aggregator/src/tools/tracing"
)

type UsecaseImports struct {
	Config         config.Config
	SessionManager transaction.SessionManager
	Usecase        Usecase
}

func NewUsecaseImports(
	log *slog.Logger,
	ri rimport.RepositoryImports,
	m *metrics.Metrics,
	tr *tracing.Tracer,
) UsecaseImports {
	// метрики не переданы (тесты, loadgen): пишем в выброшенный реестр
	if m == nil {
		m = metrics.Nop()
	}

	// трейсер не передан (тесты, loadgen): используем no-op
	if tr == nil {
		tr = tracing.Nop()
	}

	// создание блока исключенных из подсчета адресов
	internalNet, err := subnetrange.CreateDisabledSubnetRange(fmt.Sprintf("%s/%s",
		os.Getenv("SUBNET_DISABLED_DIR"), flow.InternalDisabled))
	if err != nil {
		log.Error("не удалось создать блок исключенных из подсчета адресов", "error", err)
		os.Exit(1)
	}

	// зависимостей друг от друга у этих четырех нет, порядок произволен
	flowUsecase := usecase.NewFlowUsecase(logger.NewUsecaseLogger(log, "flow"), ri)
	sessionUsecase := usecase.NewSessionUsecase(logger.NewUsecaseLogger(log, "session"), ri)
	channelUsecase := usecase.NewChannelUsecase(logger.NewUsecaseLogger(log, "channel"), ri)
	trafficUsecase := usecase.NewTrafficUsecase(
		logger.NewUsecaseLogger(log, "traffic"), ri, internalNet)

	// агрегатор собирается последним: он единственный зависит от соседей
	aggregatorUsecase := usecase.NewAggregatorUsecase(
		logger.NewUsecaseLogger(log, "aggregator"), ri,
		usecase.AggregatorDeps{
			Flow:    flowUsecase,
			Session: sessionUsecase,
			Channel: channelUsecase,
			Traffic: trafficUsecase,
		}, m, tr)

	return UsecaseImports{
		Config:         ri.Config,
		SessionManager: ri.SessionManager,

		Usecase: Usecase{
			Flow:       flowUsecase,
			Session:    sessionUsecase,
			Channel:    channelUsecase,
			Traffic:    trafficUsecase,
			Aggregator: aggregatorUsecase,
		},
	}
}
