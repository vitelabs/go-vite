FROM golang:1.15-alpine as maker

WORKDIR /go/src/github.com/vitelabs/go-vite

COPY go.mod .
COPY go.sum .

RUN GO111MODULE=on go mod download

ADD . .

RUN GO111MODULE=on go build -mod=readonly -o gvite cmd/gvite/main.go

FROM alpine:3.8

RUN apk update \
    && apk upgrade \
    && apk add --no-cache bash \
    bash-doc \
    bash-completion \
    && rm -rf /var/cache/apk/* \
    && /bin/bash

RUN apk add --no-cache ca-certificates

WORKDIR /root

COPY --from=maker /go/src/github.com/vitelabs/go-vite/gvite .
COPY --from=maker /go/src/github.com/vitelabs/go-vite/conf conf
COPY --from=maker /go/src/github.com/vitelabs/go-vite/conf/node_config.json .

EXPOSE 8483 8484 48132 41420 8483/udp
ENTRYPOINT ["./gvite"] 
