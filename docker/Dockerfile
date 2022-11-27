FROM golang:1.19.3 AS build-env
ARG NAME
ENV BASE_PATH "/go/src/${NAME}"

RUN apt-get update -qq \
    && apt-get install -yq \
                        libc6 \
                        libc6-dev \ 
                        libaio1 \ 
                        libaio-dev


COPY app ${BASE_PATH}
WORKDIR ${BASE_PATH}
ARG NAME

RUN --mount=type=cache,id=${NAME},target=/go/pkg/mod \
--mount=type=cache,id=${NAME}_cache,target=/root/.cache/go-build \ 
cd cmd && \
go build  -o ${NAME} .

FROM ubuntu:22.04

ARG NAME
ENV BASE_PATH "/go/src/${NAME}"

ENV TZ="Asia/Tashkent"
ENV DEBIAN_FRONTEND=noninteractive
RUN  apt-get update -qq \
    && apt-get install -yq \
                        libaio1 \ 
                        tzdata \
                        ca-certificates \ 
                        pkg-config && \ 
                         ln -fs /usr/share/zoneinfo/Asia/Tashkent /etc/localtime && \
    dpkg-reconfigure -f noninteractive tzdata

ENV BIN_PATH "/go/src/${NAME}"
ENV LOG_DIR "log"
ENV CONF_PATH "conf.yaml"
ENV FLOW_DIR "flow"
ENV SUBNET_DISABLED_DIR "subnet-disabled"

COPY --from=build-env ${BASE_PATH}/cmd/${NAME} /app/${NAME}

WORKDIR /app
VOLUME ["/app/log"]
VOLUME ["/app/flow"]
VOLUME ["/app/subnet-disabled"]

ARG NAME
ENV NAME_BIN=$NAME

CMD "/app/${NAME_BIN}"