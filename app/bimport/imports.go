package bimport

import "aggregator/app/internal/bridge"

type BridgeImports struct {
	Bridge Bridge
}

func (b *BridgeImports) InitBridge(
	flow bridge.Flow,
	session bridge.Session,
	traffic bridge.Traffic,
) {
	b.Bridge = Bridge{
		Flow:    flow,
		Session: session,
		Traffic: traffic,
	}
}

func NewEmptyBridge() *BridgeImports {
	return &BridgeImports{}
}
