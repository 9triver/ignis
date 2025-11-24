FROM golang:1.25.4 AS base

RUN sed -i 's/deb.debian.org/mirrors.ustc.edu.cn/g' /etc/apt/sources.list.d/debian.sources

RUN apt-get update && \
  apt-get install -y build-essential libczmq-dev pkg-config

WORKDIR /build

COPY ./ignis-go /build

RUN go build -o main cmd/main.go

ENTRYPOINT [ "/build/main" ]
