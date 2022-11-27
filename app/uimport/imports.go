package uimport

import (
	"aggregator/app/bimport"
	"aggregator/app/config"
	"aggregator/app/internal/entity/global"
	"aggregator/app/internal/transaction"
	"aggregator/app/internal/usecase"
	"aggregator/app/rimport"
	"aggregator/app/tools/subnetrange"
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
	sessionManager transaction.SessionManager,
) UsecaseImports {
	config, err := config.NewConfig(os.Getenv("CONF_PATH"))
	if err != nil {
		log.Fatalln(err)
	}

	// создание блока исключенных из подсчета адресов
	internalNet, err := subnetrange.CreateDisabledSubnetRange(fmt.Sprintf("%s/%s",
		os.Getenv("SUBNET_DISABLED_DIR"), global.InternalDisabled))
	if err != nil {
		log.Fatalln(err)
	}

	ui := UsecaseImports{
		Config:         config,
		SessionManager: sessionManager,

		Usecase: Usecase{
			Flow:       usecase.NewFlowUsecase(log, ri),
			Session:    usecase.NewSessionUsecase(log, ri),
			Traffic:    usecase.NewTrafficUsecase(log, ri, bi, internalNet),
			Aggregator: usecase.NewAggregatorUsecase(log, ri, bi),
		},
		BridgeImports: bi,
	}

	return ui
}
