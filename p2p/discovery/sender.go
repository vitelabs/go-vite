package discovery

import (
	"net"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/p2p2/vnode"

	"github.com/vitelabs/go-vite/crypto/ed25519"
)

type Sender interface {
	ping(addr string) (echo types.Hash)
}

type sender struct {
	privateKey ed25519.PrivateKey
	self       *vnode.Node
	conn       *net.UDPConn
}

func (s *sender) send(msg, to *net.UDPAddr) {

}

func (s *sender) ping(addr string) (echo types.Hash) {

}

func (s *sender) pong(echo types.Hash, to *net.UDPAddr) {

}

func (s *sender) findnode(target vnode.NodeID, to *net.UDPAddr) {

}

func (s *sender) sendNodes(eps []vnode.EndPoint, to *net.UDPAddr) {

}
