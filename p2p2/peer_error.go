package p2p

type PeerError byte

const (
	PeerNetworkError PeerError = iota // read/write timeout, read/write error
	PeerProtocolError
	PeerNoMatchedProtocols
	PeerDifferentNetwork
	PeerTooManyPeers
	PeerTooManySameNetPeers
	PeerTooManyInboundPeers
	PeerAlreadyConnected
	PeerIncompatibleVersion
	PeerQuitting
	PeerResponseTimeout
	PeerNotHandshakeMsg
	PeerInvalidSignature
	PeerConnectSelf
	PeerUnknownProtocol
	PeerUnknownCode
)

var peerErrStr = [...]string{
	PeerNetworkError:        "network error",
	PeerProtocolError:       "protocol error",
	PeerNoMatchedProtocols:  "no matched protocols",
	PeerDifferentNetwork:    "different network",
	PeerTooManyPeers:        "too many peers",
	PeerTooManySameNetPeers: "too many peers in the same net",
	PeerTooManyInboundPeers: "too many inbound peers",
	PeerAlreadyConnected:    "already connected",
	PeerIncompatibleVersion: "incompatible p2p version",
	PeerConnectSelf:         "connected to self",
	PeerResponseTimeout:     "response timeout",
	PeerNotHandshakeMsg:     "not handshake message",
	PeerQuitting:            "client quitting",
	PeerInvalidSignature:    "invalid signature",
	PeerUnknownProtocol:     "unknown protocol",
	PeerUnknownCode:         "unknown message code",
}

func (e PeerError) String() string {
	if int(e) < len(peerErrStr) {
		return peerErrStr[e]
	}
	return "unknown error"
}

func (e PeerError) Error() string {
	return e.String()
}

func (e PeerError) Serialize() []byte {
	return []byte{
		byte(e),
	}
}

func deserialize(buf []byte) PeerError {
	return PeerError(buf[0])
}
