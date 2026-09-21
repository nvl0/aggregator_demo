SHELL := /bin/bash

SRC := $(CURDIR)/src
SCRIPTS := $(SRC)/scripts

# те же значения, что и в .github/workflows/ci.yml, но абсолютными путями
export CONF_PATH ?= $(SRC)/config/conf.yaml
export FLOW_DIR ?= $(SRC)/flow
export SUBNET_DISABLED_DIR ?= $(SRC)/subnet-disabled
export PG_URL ?= localhost:5432
export DEBUG ?= true

GOTEST := go test -race -p 1
GOLANGCI_VERSION := v2.13.2

.PHONY: help lint test test-unit test-storage test-pg build docker-build docker-up db-up jaeger-up mocks loadgen

help: ## список таргетов
	@grep -E '^[a-z-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "%-14s %s\n", $$1, $$2}'

lint: ## golangci-lint по всему модулю (docker, версия как в CI)
	docker run --rm -v $(CURDIR):/app -w /app/src \
		golangci/golangci-lint:$(GOLANGCI_VERSION) golangci-lint run

test-unit: ## юнит-тесты usecase, tools (кроме gensql/pgdb - им нужен postgres) и config
	cd $(SRC) && $(GOTEST) ./internal/usecase/test/... ./tools/dump/... ./tools/flowgen/... \
		./tools/logger/... ./tools/measure/... ./tools/metrics/... ./tools/sqlnull/... \
		./tools/subnetrange/... ./tools/tracing/... ./tools/workerpool/... ./external/... ./config/...

test-storage: ## тесты файлового репозитория
	cd $(SRC) && $(GOTEST) ./internal/repository/storage/test/...

test-pg: db-up ## интеграционные тесты postgresql репозитория, transaction, tools/gensql и tools/pgdb (нужен docker)
	cd $(SRC) && $(GOTEST) ./internal/repository/postgresql/test/... ./internal/transaction/test/... \
		./tools/gensql/... ./tools/pgdb/...

test: test-unit test-storage test-pg ## все тесты подряд

build: ## сборка бинарника в bin/
	cd $(SRC)/cmd && go build -o $(CURDIR)/bin/aggregator .

docker-build: ## сборка docker образа
	cd $(SCRIPTS) && ./build_docker.sh

docker-up: ## поднять весь стек (db + migrate + core)
	cd $(SCRIPTS) && ./start_docker.sh

db-up: ## поднять только бд и накатить миграции
	docker compose -p aggregator -f $(CURDIR)/docker/docker-compose.yaml up -d db migrate

jaeger-up: ## поднять jaeger (UI localhost:16686, OTLP/HTTP :4318)
	docker compose -p aggregator -f $(CURDIR)/docker/docker-compose.yaml up -d jaeger

mocks: ## регенерация моков
	cd $(SCRIPTS) && ./mocks_creator.sh

loadgen: ## dev-замер масштабируемости агрегации
	cd $(SCRIPTS) && ./run_loadgen.sh
