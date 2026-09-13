#!/usr/bin/env bash

mockgen -destination=../internal/repository/mocks.go -package=repository -source=../internal/repository/interface.go
mockgen -destination=../internal/transaction/mocks.go -package=transaction -source=../internal/transaction/interface.go
mockgen -destination=../internal/usecase/mocks.go -package=usecase -source=../internal/usecase/interface.go
