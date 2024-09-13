FROM golang:1.23.1 AS build-env
ARG NAME
ENV BASE_PATH "/go/src/${NAME}"

COPY src ${BASE_PATH}
WORKDIR ${BASE_PATH}
ARG NAME

RUN --mount=type=cache,id=${NAME},target=/go/pkg/mod \
--mount=type=cache,id=${NAME}_cache,target=/root/.cache/go-build \ 
cd cmd && \
go build -o ${NAME} .

FROM ubuntu:24.04
ARG NAME
ENV BASE_PATH "/go/src/${NAME}"

ENV TZ="Asia/Tashkent"
ENV DEBIAN_FRONTEND=noninteractive
RUN  apt-get update -qq \
    && apt-get install -yq \
                        tzdata \
                        apparmor-utils \
                        ca-certificates \
                        curl && \
                        ln -fs /usr/share/zoneinfo/Asia/Tashkent /etc/localtime && \
    dpkg-reconfigure -f noninteractive tzdata && \
    rm -rf /var/lib/apt/lists/*;

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