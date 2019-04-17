package booter

import (
	"net"

	"github.com/vitelabs/go-vite/p2p/discovery"
)

type server struct {
	discv discovery.Discovery
	ln    net.Listener
	term  chan struct{}
}

func (s *server) start() {

}

func (s *server) stop() {

}
