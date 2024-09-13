#!/usr/bin/env bash

export ROOT=../..
source variables.sh

mkdir -p $ROOT/log
mkdir -p $ROOT/bin

echo 'BUILD DOCKER'
cd $ROOT/docker
docker compose build 
