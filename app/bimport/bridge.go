package bimport

import "aggregator/app/internal/bridge"

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
