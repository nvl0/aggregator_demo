package uimport

import (
	"aggregator/src/bimport"
	"aggregator/src/config"
	"aggregator/src/internal/entity/global"
	"aggregator/src/internal/transaction"
	"aggregator/src/internal/usecase"
	"aggregator/src/rimport"
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
	disabledNet, err := subnetrange.CreateDisabledSubnetRange(fmt.Sprintf("%s/%s",
		os.Getenv("SUBNET_DISABLED_DIR"), global.InternalDisabled))
	if err != nil {
		log.Fatalln(err)
	}

	ui := UsecaseImports{
		Config:         config,
		SessionManager: ri.SessionManager,

		Usecase: Usecase{
			Flow:       usecase.NewFlowUsecase(log, ri),
			Session:    usecase.NewSessionUsecase(log, ri),
			Traffic:    usecase.NewTrafficUsecase(log, ri, bi, disabledNet),
			Aggregator: usecase.NewAggregatorUsecase(log, ri, bi),
		},
		BridgeImports: bi,
	}

	return ui
}
