#!/usr/bin/env bash

mockgen -destination=../internal/repository/mocks.go -package=repository -source=../internal/repository/interface.go
mockgen -destination=../internal/bridge/mocks.go -package=bridge -source=../internal/bridge/interface.go
mockgen -destination=../internal/transaction/mocks.go -package=transaction -source=../internal/transaction/interface.go
