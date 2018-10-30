FROM golang:1.11-alpine

ADD . /vite
RUN cd /vite && go build -o gvite  github.com/vitelabs/go-vite/cmd/gvite
