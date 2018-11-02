FROM golang:1.11-alpine as maker

RUN set -eux; \
 	apk add gcc \
 	    musl-dev

ADD . /usr/local/go/src/github.com/vitelabs/go-vite
RUN go build -o gvite  github.com/vitelabs/go-vite/cmd/gvite

FROM alpine:3.8
COPY --from=maker /go/gvite .
COPY ./node_config.json .
EXPOSE 8483 8484 48132 41420 8483/udp
ENTRYPOINT ["/gvite"]
