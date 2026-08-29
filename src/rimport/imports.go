package rimport

import (
	"aggregator/src/config"
	"aggregator/src/internal/repository/postgresql"
	"aggregator/src/internal/repository/storage"
	"aggregator/src/internal/transaction"
	"log/slog"
	"os"
)

// newFatalLogger локальный логгер для ошибок инициализации до появления основного логгера приложения
func newFatalLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}

type RepositoryImports struct {
	Config         config.Config
	SessionManager transaction.SessionManager
	Repository     Repository
}

// NewRepositoryImports собирает репозитории. Конфиг передается снаружи:
// он уже загружен в main и повторный парсинг файла не нужен.
func NewRepositoryImports(
	conf config.Config,
	sessionManager transaction.SessionManager,
) RepositoryImports {
	return RepositoryImports{
		Config:         conf,
		SessionManager: sessionManager,
		Repository: Repository{
			Session: postgresql.NewSessionRepository(),
			Channel: postgresql.NewChannelRepository(),
			Flow: storage.NewFlowRepository(os.Getenv("FLOW_DIR"),
				os.Getenv("SUBNET_DISABLED_DIR")),
			FlowBatch: postgresql.NewFlowBatchRepository(),
		},
	}
}
