package discovery

import (
	"errors"
	"fmt"
	"time"

	"github.com/vitelabs/go-vite/p2p2/vnode"
)

var errInvalidEndPointBytes = errors.New("invalid endpoint bytes")
var errMissingHost = errors.New("missing host")
var errHostTooLong = fmt.Errorf("host should be short than %d bytes", maxHostLength)

type code byte

const (
	pingCode    code = iota // ping a node to check it alive, also announce self, expect to get a pingAck or pong
	pingAckCode             // pingAck is response to ping and check peer`s auth, expect to get a pong
	pongCode                // response to ping/pingAck
	findNodeCode
	neighborsCode
)

const messageTimeout = 5 * time.Second

const maxHostLength = 1<<7 - 1

type ping struct {
	from      vnode.Node
	to        endPoint
	timestamp int64
	nonce     []byte
	ext       []byte
}

type pong struct {
	id        vnode.NodeID
	to        endPoint
	timestamp int64
	ext       []byte
	ack       []byte
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
	neighbors []endPoint
}
