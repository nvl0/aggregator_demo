#!/bin/bash
export DEBUG=true
export CONF_PATH=../../../../../config/conf.yaml
export FLOW_DIR=../../../../../flow
export SUBNET_DISABLED_DIR=../../../../../subnet-disabled


if [ -z "$1" ]
  then
    echo "Укажите название репозитория"
    exit 1
fi

MODULE=$1
FUNC_NAME=""

if [ -n "$2" ]
  then
    FUNC_NAME="-v --run $2"
    echo $FUNC_NAME
fi

cd ../internal/repository/storage/test/$MODULE
go test $FUNC_NAME