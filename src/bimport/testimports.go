package bimport

import (
	"aggregator/src/internal/bridge"

	"go.uber.org/mock/gomock"
)

type TestBridgeImports struct {
	ctrl       *gomock.Controller
	TestBridge TestBridge
}

func NewTestBridgeImports(
	ctrl *gomock.Controller,
) *TestBridgeImports {
	return &TestBridgeImports{
		ctrl: ctrl,
		TestBridge: TestBridge{
			Flow:       bridge.NewMockFlow(ctrl),
			Session:    bridge.NewMockSession(ctrl),
			Channel:    bridge.NewMockChannel(ctrl),
			Traffic:    bridge.NewMockTraffic(ctrl),
			Aggregator: bridge.NewMockAggregator(ctrl),
		},
	}
}

func (t *TestBridgeImports) BridgeImports() *BridgeImports {
	return &BridgeImports{
		Bridge: Bridge{
			Flow:       t.TestBridge.Flow,
			Session:    t.TestBridge.Session,
			Channel:    t.TestBridge.Channel,
			Traffic:    t.TestBridge.Traffic,
			Aggregator: t.TestBridge.Aggregator,
		},
	}
}
