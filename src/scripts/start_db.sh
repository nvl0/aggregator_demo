#!/usr/bin/env bash

export ROOT=..
source variables.sh

cd $ROOT/../docker
docker compose -p aggregator up --force-recreate --remove-orphans