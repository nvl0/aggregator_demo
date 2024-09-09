package rimport

import (
	"aggregator/src/config"
	"aggregator/src/internal/repository"
	"aggregator/src/internal/transaction"
	"log"
	"os"

	"go.uber.org/mock/gomock"
)

type TestRepositoryImports struct {
	Config         config.Config
	SessionManager *transaction.MockSessionManager
	MockRepository MockRepository
	ctrl           *gomock.Controller
}

func NewTestRepositoryImports(
	ctrl *gomock.Controller,
) TestRepositoryImports {
	config, err := config.NewConfig(os.Getenv("CONF_PATH"))
	if err != nil {
		log.Fatalln(err)
	}

	return TestRepositoryImports{
		ctrl:           ctrl,
		Config:         config,
		SessionManager: transaction.NewMockSessionManager(ctrl),
		MockRepository: MockRepository{
			Session: repository.NewMockSession(ctrl),
			Channel: repository.NewMockChannel(ctrl),
			Flow:    repository.NewMockFlow(ctrl),
		},
	}
}

func (t *TestRepositoryImports) RepositoryImports() RepositoryImports {
	return RepositoryImports{
		SessionManager: t.SessionManager,
		Config:         t.Config,
		Repository: Repository{
			Session: t.MockRepository.Session,
			Channel: t.MockRepository.Channel,
			Flow:    t.MockRepository.Flow,
		},
	}
}
