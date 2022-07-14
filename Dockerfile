FROM golang:1.18 as build
WORKDIR $GOPATH/src/github.com/vkumbhar94/lm-bootstrap-collector
ARG VERSION
COPY ./ ./
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /lmbc

FROM ubuntu:21.10
ENV DEBIAN_FRONTEND noninteractive

# NTP is needed for some collector operations
RUN apt update && apt-get update \
  && apt install software-properties-common -y \
  && add-apt-repository ppa:deadsnakes/ppa \
  && apt-get install --no-install-recommends -y \
  tcl \
  inetutils-traceroute \
  file \
  iputils-ping \
  ntp \
  perl \
  procps \
  xxd \
#  python3.10 \
#  python3-pip \
  && apt-get -y clean \
  && rm -rf /var/lib/apt/lists/*
#  && ln -s /usr/bin/python3.10 /usr/bin/python \
#  && pip config set global.target /usr/local/lib/python3.10/dist-packages

COPY --from=build /lmbc /bin
RUN #pip install --no-cache-dir logicmonitor_sdk==1.0.129
RUN mkdir /usr/local/logicmonitor

COPY ./entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
