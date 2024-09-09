package rimport

import (
	"aggregator/src/config"
	"aggregator/src/internal/repository/postgresql"
	"aggregator/src/internal/repository/storage"
	"aggregator/src/internal/transaction"
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
			Channel: postgresql.NewChannelRepository(),
			Flow: storage.NewFlowRepository(os.Getenv("FLOW_DIR"),
				os.Getenv("SUBNET_DISABLED_DIR")),
		},
	}
}
