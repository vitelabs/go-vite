package p2p

// Protocol is a abstract communication layer above connection, there can be multi protocol on a connection.
// A Protocol usually has many different message codes, each code has a handler to handle the message from peer.
type Protocol interface {
	// Name is the protocol name, should better be unique, for readability
	Name() string

	// ID MUST be unique
	ID() ProtocolID

	// Auth encrypt the input, return the output
	// output will be sent to peer handshake
	Auth(input []byte) (output []byte)

	// Handshake handle the peer handshakeData, if the return error is not nil, will disconnect with peer
	Handshake(their []byte) error

	// Handle message from sender, if the return error is not nil, will disconnect with peer
	Handle(msg Msg) error

	// OnPeerAdded will be invoked after Peer run
	// peer will be closed if return error is not nil
	OnPeerAdded(peer Peer) error

	// OnPeerRemoved will be invoked after Peer closed
	OnPeerRemoved(peer Peer) error
}

type MsgHandler = func(msg Msg) error

type ProtocolMap = map[ProtocolID]Protocol
