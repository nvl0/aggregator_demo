package deps_test

import (
	"testing"

	"aggregator/src/internal/usecase"

	"github.com/stretchr/testify/require"
)

// TestUsecaseImplementsDeps конкретные usecase удовлетворяют узким интерфейсам
// соседей, которые агрегатор получает через AggregatorDeps
func TestUsecaseImplementsDeps(t *testing.T) {
	tests := []struct {
		name string
		// ifacePtr нулевой указатель на интерфейс: так testify узнает тип интерфейса
		ifacePtr any
		impl     any
	}{
		{
			name:     "FlowUsecase реализует FlowStreamer",
			ifacePtr: (*usecase.FlowStreamer)(nil),
			impl:     new(usecase.FlowUsecase),
		},
		{
			name:     "SessionUsecase реализует SessionLoader",
			ifacePtr: (*usecase.SessionLoader)(nil),
			impl:     new(usecase.SessionUsecase),
		},
		{
			name:     "ChannelUsecase реализует ChannelLoader",
			ifacePtr: (*usecase.ChannelLoader)(nil),
			impl:     new(usecase.ChannelUsecase),
		},
		{
			name:     "TrafficUsecase реализует TrafficProcessor",
			ifacePtr: (*usecase.TrafficProcessor)(nil),
			impl:     new(usecase.TrafficUsecase),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.New(t).Implements(tt.ifacePtr, tt.impl)
		})
	}
}
