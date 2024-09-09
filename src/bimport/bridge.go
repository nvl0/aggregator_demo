package bimport

import "aggregator/src/internal/bridge"

type Bridge struct {
	Flow    bridge.Flow
	Session bridge.Session
	Channel bridge.Channel
	Traffic bridge.Traffic
}

type TestBridge struct {
	Flow    *bridge.MockFlow
	Session *bridge.MockSession
	Channel *bridge.MockChannel
	Traffic *bridge.MockTraffic
}
