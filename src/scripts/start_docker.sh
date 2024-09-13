#!/usr/bin/env bash

export ROOT=../..
source variables.sh

./build_docker.sh

echo 'RUN DOCKER'
cd $ROOT/docker
docker compose -p aggregator up $1 --force-recreate --remove-orphans
