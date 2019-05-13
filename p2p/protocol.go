package p2p

import (
	"github.com/vitelabs/go-vite/common/types"
)

// Protocol is a abstract communication layer above connection, there can be multi protocol on a connection.
// A Protocol usually has many different message codes, each code has a handler to handle the message from peer.
type Protocol interface {
	// ProtoData return the data to handshake, will transmit to peer with HandshakeMsg.
	ProtoData() (key []byte, height uint64, genesis types.Hash)

	// ReceiveHandshake handle the HandshakeMsg and protoData from peer.
	// The connection will be disconnected if err is not nil.
	// Level MUST small than 255, each level has it`s strategy, Eg, access by count-constraint, broadcast priority
	// If peer support multiple protocol, the highest protocol level will be the peer`s level.
	// As default, there are 4 levels from low to high: Inbound - Outbound - Trusted - Superior
	ReceiveHandshake(msg *HandshakeMsg) (level Level, err error)

	// Handle message from sender, if the return error is not nil, will disconnect with peer
	Handle(msg Msg) error

	// State get the Protocol state, will be sent to peers by heartbeat
	State() []byte

	// OnPeerAdded will be invoked after Peer run
	// peer will be closed if return error is not nil
	OnPeerAdded(peer Peer) error

	// OnPeerRemoved will be invoked after Peer closed
	OnPeerRemoved(peer Peer) error
}

type MsgHandler = func(msg Msg) error

type Level = byte

const (
	Inbound  Level = 0
	Outbound Level = 1
	Trusted  Level = 2
	Superior Level = 3
)
