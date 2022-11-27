package rimport

import (
	"aggregator/app/config"
	"aggregator/app/internal/repository/postgresql"
	"aggregator/app/internal/repository/storage"
	"aggregator/app/internal/transaction"
	"log"
	"os"
)

type RepositoryImports struct {
	Config         config.Config
	SessionManager transaction.SessionManager
	Repository     Repository
}

func NewRepositoryImports(
	sessionManager transaction.SessionManager,
) RepositoryImports {
	conf, err := config.NewConfig(os.Getenv("CONF_PATH"))
	if err != nil {
		log.Fatalln(err)
	}

	return RepositoryImports{
		Config:         conf,
		SessionManager: sessionManager,
		Repository: Repository{
			Session: postgresql.NewSessionRepository(),
			Flow: storage.NewFlowRepository(os.Getenv("FLOW_DIR"),
				os.Getenv("SUBNET_DISABLED_DIR")),
		},
	}
}
