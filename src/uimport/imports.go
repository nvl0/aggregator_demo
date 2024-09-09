package uimport

import (
	"aggregator/src/bimport"
	"aggregator/src/config"
	"aggregator/src/internal/entity/flow"
	"aggregator/src/internal/transaction"
	"aggregator/src/internal/usecase"
	"aggregator/src/rimport"
	"aggregator/src/tools/logger"
	"aggregator/src/tools/subnetrange"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

type UsecaseImports struct {
	Config         config.Config
	SessionManager transaction.SessionManager
	Usecase        Usecase
	*bimport.BridgeImports
}

func NewUsecaseImports(
	log *logrus.Logger,
	ri rimport.RepositoryImports,
	bi *bimport.BridgeImports,
) UsecaseImports {
	config, err := config.NewConfig(os.Getenv("CONF_PATH"))
	if err != nil {
		log.Fatalln(err)
	}

	// создание блока исключенных из подсчета адресов
	internalNet, err := subnetrange.CreateDisabledSubnetRange(fmt.Sprintf("%s/%s",
		os.Getenv("SUBNET_DISABLED_DIR"), flow.InternalDisabled))
	if err != nil {
		log.Fatalln(err)
	}

	ui := UsecaseImports{
		Config:         config,
		SessionManager: ri.SessionManager,

		Usecase: Usecase{
			Flow:       usecase.NewFlowUsecase(logger.NewUsecaseLogger(log, "flow"), ri),
			Session:    usecase.NewSessionUsecase(logger.NewUsecaseLogger(log, "session"), ri),
			Channel:    usecase.NewChannelUsecase(logger.NewUsecaseLogger(log, "channel"), ri),
			Traffic:    usecase.NewTrafficUsecase(logger.NewUsecaseLogger(log, "traffic"), ri, bi, internalNet),
			Aggregator: usecase.NewAggregatorUsecase(logger.NewUsecaseLogger(log, "aggregator"), ri, bi),
		},
		BridgeImports: bi,
	}

	return ui
}
