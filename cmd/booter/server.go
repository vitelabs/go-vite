package main

import (
	"net"

	"github.com/vitelabs/go-vite/p2p/discovery"
)

type server struct {
	discv discovery.Discovery
	ln    net.Listener
	term  chan struct{}
}

func newServer() *server {

	return nil
}

func (s *server) start() {

}

func (s *server) stop() {

}
