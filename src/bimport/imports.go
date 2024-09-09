package bimport

import "aggregator/src/internal/bridge"

type BridgeImports struct {
	Bridge Bridge
}

func (b *BridgeImports) InitBridge(
	flow bridge.Flow,
	session bridge.Session,
	channel bridge.Channel,
	traffic bridge.Traffic,
) {
	b.Bridge = Bridge{
		Flow:    flow,
		Session: session,
		Channel: channel,
		Traffic: traffic,
	}
}

func NewEmptyBridge() *BridgeImports {
	return &BridgeImports{}
}
