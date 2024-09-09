#!/usr/bin/env bash

export ROOT=../..
source variables.sh

mv ../flow/127.0.0.0/tmp/ft-test_data ../flow/127.0.0.0/

mkdir -p $ROOT/log
mkdir -p $ROOT/bin

echo 'RUN DOCKER'

cd $ROOT/docker
docker compose build 
docker compose up --force-recreate --remove-orphans
