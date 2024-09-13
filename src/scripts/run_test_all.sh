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

echo '-- тест storage repository --'
export CONF_PATH=../../../../../config/conf.yaml
export FLOW_DIR=../../../../../flow
export SUBNET_DISABLED_DIR=../../../../../subnet-disabled
for s in $(go list ./repository/storage/test/...); do if ! go test -failfast -p 1 $s; then break; fi; done 2>&1

echo '-- тест pg repository --'
echo '-- проверка docker --'
DOCKOUT=$(docker ps -af name=aggregator-db-1 --format "{{.State}}")

if [[ "$DOCKOUT" = "" || "$DOCKOUT" = "created" ]]
    then
        echo "Сборка docker контейнера ранее не осуществлялась. Начинаю собирать docker..."
        cd ../scripts/
        ./build_docker.sh
        ./start_docker.sh -d
        docker stop aggregator-core-1
        cd ../internal/
fi

if [[ "$DOCKOUT" = "exited" || "$DOCKOUT" = "paused" ]] 
    then 
        docker start aggregator-db-1
elif [[ "$DOCKOUT" = "restarting" ]] 
    then
        echo "Контейнер с бд перезапускается. Пожалуйста подождите"
        sleep 5
elif [[ "$DOCKOUT" = "removing" || "$DOCKOUT" = "dead" ]]
    then
        echo "Нет возможности запустить тесты pg из-за статуса контейнера $DOCKOUT"
        exit 1
fi

for (( ; ; ))
do
    DOCKOUT=$(docker ps -af name=aggregator-db-1 --format "{{.State}}")
    if [[ "$DOCKOUT" = "running" ]]
        then
            sleep 5
            break
    fi
    sleep 2
done

for s in $(go list ./repository/postgresql/test/...); do if ! go test -failfast -p 1 $s; then break; fi; done 2>&1

sleep 5
docker stop aggregator-db-1
