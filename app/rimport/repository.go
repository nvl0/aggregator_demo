package rimport

import "aggregator/app/internal/repository"

type Repository struct {
	Session repository.Session
	Flow    repository.Flow
}

type MockRepository struct {
	Session *repository.MockSession
	Flow    *repository.MockFlow
}
