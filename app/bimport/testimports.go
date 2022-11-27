package bimport

import (
	"aggregator/app/internal/bridge"

	"github.com/golang/mock/gomock"
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
			Flow:    bridge.NewMockFlow(ctrl),
			Session: bridge.NewMockSession(ctrl),
			Traffic: bridge.NewMockTraffic(ctrl),
		},
	}
}

func (t *TestBridgeImports) BridgeImports() *BridgeImports {
	return &BridgeImports{
		Bridge: Bridge{
			Flow:    t.TestBridge.Flow,
			Session: t.TestBridge.Session,
			Traffic: t.TestBridge.Traffic,
		},
	}
}
