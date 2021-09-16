FROM golang:1.13-alpine as maker

RUN set -eux; \
    apk add gcc \
    musl-dev \
    git
WORKDIR /go/src/github.com/vitelabs/go-vite

COPY go.mod .
COPY go.sum .

RUN GO111MODULE=on go mod download

ADD . /go/src/github.com/vitelabs/go-vite

RUN GO111MODULE=on go build -mod=readonly -o gvite github.com/vitelabs/go-vite/cmd/gvite

FROM alpine:3.8

RUN apk update \
    && apk upgrade \
    && apk add --no-cache bash \
    bash-doc \
    bash-completion \
    && rm -rf /var/cache/apk/* \
    && /bin/bash

RUN apk add --no-cache ca-certificates

COPY --from=maker /go/src/github.com/vitelabs/go-vite/gvite .
COPY ./conf/node_config.json .
COPY ./docker/docker-gvite .
EXPOSE 8483 8484 48132 41420 8483/udp
ENTRYPOINT ["/docker-gvite"] 
