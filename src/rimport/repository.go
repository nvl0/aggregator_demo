package rimport

import "aggregator/src/internal/repository"

type Repository struct {
	Session   repository.Session
	Channel   repository.Channel
	Flow      repository.Flow
	FlowBatch repository.FlowBatch
}

type MockRepository struct {
	Session   *repository.MockSession
	Channel   *repository.MockChannel
	Flow      *repository.MockFlow
	FlowBatch *repository.MockFlowBatch
}
