#!/usr/bin/env bash

export ROOT=..
source variables.sh

mkdir -p $ROOT/log
mkdir -p $ROOT/bin

echo 'RUN DOCKER'

cd $ROOT/../docker
docker compose build 
docker compose up --force-recreate --remove-orphans
