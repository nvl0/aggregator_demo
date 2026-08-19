#!/bin/bash
# dev-инструмент замера масштабируемости агрегации
# требуется запущенный контейнер aggregator-db-1

export DEBUG=false
export MEASURE=disable
# FLOWGEN читается пакетом usecase при инициализации,
# генерацией flow здесь занимается сам loadgen
export FLOWGEN=false
export CONF_PATH=../../config/conf.yaml
export FLOW_DIR=../../flow
export SUBNET_DISABLED_DIR=../../subnet-disabled
export PG_URL=localhost:5432

cd ../cmd/loadgen
go run . "$@"
