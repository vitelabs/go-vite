FROM golang:1.11-alpine as maker

ADD . /usr/local/go/src/github.com/vitelabs/go-vite
RUN set -eux; \
 	apk add --no-cache --virtual .build-deps \
 	    gcc

RUN go build -o gvite  github.com/vitelabs/go-vite/cmd/gvite

FROM alpine:latest
COPY --from=maker ./gvite .
COPY ./node_config.json .
EXPOSE 8483 8483/udp 8484 48132 41420
ENTRYPOINT ["gvite"]
