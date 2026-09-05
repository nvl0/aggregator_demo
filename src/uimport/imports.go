package uimport

import (
	"fmt"
	"log/slog"
	"os"

	"aggregator/src/bimport"
	"aggregator/src/config"
	"aggregator/src/internal/entity/flow"
	"aggregator/src/internal/transaction"
	"aggregator/src/internal/usecase"
	"aggregator/src/rimport"
	"aggregator/src/tools/logger"
	"aggregator/src/tools/metrics"
	"aggregator/src/tools/subnetrange"
)

type UsecaseImports struct {
	Config         config.Config
	SessionManager transaction.SessionManager
	Usecase        Usecase
	*bimport.BridgeImports
}

func NewUsecaseImports(
	log *slog.Logger,
	ri rimport.RepositoryImports,
	bi *bimport.BridgeImports,
	m *metrics.Metrics,
) UsecaseImports {
	// метрики не переданы (тесты, loadgen): пишем в выброшенный реестр
	if m == nil {
		m = metrics.Nop()
	}

	// создание блока исключенных из подсчета адресов
	internalNet, err := subnetrange.CreateDisabledSubnetRange(fmt.Sprintf("%s/%s",
		os.Getenv("SUBNET_DISABLED_DIR"), flow.InternalDisabled))
	if err != nil {
		log.Error("не удалось создать блок исключенных из подсчета адресов", "error", err)
		os.Exit(1)
	}

	ui := UsecaseImports{
		Config:         ri.Config,
		SessionManager: ri.SessionManager,

		Usecase: Usecase{
			Flow:       usecase.NewFlowUsecase(logger.NewUsecaseLogger(log, "flow"), ri),
			Session:    usecase.NewSessionUsecase(logger.NewUsecaseLogger(log, "session"), ri),
			Channel:    usecase.NewChannelUsecase(logger.NewUsecaseLogger(log, "channel"), ri),
			Traffic:    usecase.NewTrafficUsecase(logger.NewUsecaseLogger(log, "traffic"), ri, bi, internalNet),
			Aggregator: usecase.NewAggregatorUsecase(logger.NewUsecaseLogger(log, "aggregator"), ri, bi, m),
		},
		BridgeImports: bi,
	}

	return ui
}
