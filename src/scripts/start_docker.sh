#!/usr/bin/env bash

export ROOT=../..
source variables.sh

mv ../flow/127.0.0.0/tmp/ft-test_data ../flow/127.0.0.0/

./build_docker.sh

echo 'RUN DOCKER'
cd $ROOT/docker
docker compose -p aggregator up $1 --force-recreate --remove-orphans
