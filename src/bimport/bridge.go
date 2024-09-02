package bimport

import "aggregator/src/internal/bridge"

type Bridge struct {
	Flow    bridge.Flow
	Session bridge.Session
	Traffic bridge.Traffic
}

type TestBridge struct {
	Flow    *bridge.MockFlow
	Session *bridge.MockSession
	Traffic *bridge.MockTraffic
}
