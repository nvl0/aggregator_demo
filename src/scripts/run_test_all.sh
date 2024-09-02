#!/bin/bash
export ROOT=../../..
source variables.sh
export DEBUG=true
export DEV=true
export FLOW_DIR=../../../../flow
export SUBNET_DISABLED_DIR=../../../../subnet-disabled

go clean -testcache
cd ../internal/
echo '-- тест usecase --'
export CONF_PATH=../../../../config/conf.yaml
for s in $(go list ./usecase/test/...); do if ! go test -failfast -p 1 $s; then break; fi; done 2>&1
echo '-- тест repository --'
export CONF_PATH=../../../../../config/conf.yaml
for s in $(go list ./repository/postgresql/test/...); do if ! go test -failfast -p 1 $s; then break; fi; done 2>&1
export FLOW_DIR=../../../../../flow
export SUBNET_DISABLED_DIR=../../../../../subnet-disabled
for s in $(go list ./repository/storage/test/...); do if ! go test -failfast -p 1 $s; then break; fi; done 2>&1