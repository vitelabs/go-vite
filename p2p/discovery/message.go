package discovery

import (
	"net"
	"time"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/p2p2/vnode"
)

type code byte

const (
	pingCode    code = iota // ping a node to check it alive, also announce self, expect to get a pingAck or pong
	pingAckCode             // pingAck is response to ping and check peer`s auth, expect to get a pong
	pongCode                // response to ping/pingAck
	findNodeCode
	neighborsCode
)

type ingressMsg struct {
	c    code
	from *net.UDPAddr
	body interface{}
}

type msg interface {
	code() code
	serialize() []byte
}

const messageTimeout = 5 * time.Second
const maxNodesToFind = 200

type ping struct {
	from      vnode.Node
	to        vnode.EndPoint
	timestamp int64
}

type pong struct {
	from      vnode.Node
	to        vnode.EndPoint
	timestamp int64
	ack       types.Hash
}

type findNode struct {
	id        vnode.NodeID
	target    vnode.NodeID
	count     int
	timestamp int64
}

type neighbors struct {
	id        vnode.NodeID
	timestamp int
	neighbors []vnode.EndPoint
}

func paseMessage(pkt packet) (c code, from *net.UDPAddr, data interface{}, err error) {

}
