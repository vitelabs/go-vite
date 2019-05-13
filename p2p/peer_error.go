package p2p

type PeerError byte

const (
	PeerNetworkError PeerError = iota // read/write timeout, read/write error
	PeerDifferentNetwork
	PeerTooManyPeers
	PeerTooManySameNetPeers
	PeerTooManyInboundPeers
	PeerAlreadyConnected
	PeerIncompatibleVersion
	PeerQuitting
	PeerNotHandshakeMsg
	PeerInvalidSignature
	PeerConnectSelf
	PeerUnknownMessage
	PeerUnmarshalError
	PeerNoPermission
	PeerBanned
	PeerDifferentGenesis
	PeerInvalidBlock
	PeerInvalidMessage
)

var peerErrStr = [...]string{
	PeerNetworkError:        "network error",
	PeerDifferentNetwork:    "different network",
	PeerTooManyPeers:        "too many peers",
	PeerTooManySameNetPeers: "too many peers in the same net",
	PeerTooManyInboundPeers: "too many inbound peers",
	PeerAlreadyConnected:    "already connected",
	PeerIncompatibleVersion: "incompatible p2p version",
	PeerConnectSelf:         "connected to self",
	PeerNotHandshakeMsg:     "not handshake message",
	PeerQuitting:            "client quitting",
	PeerInvalidSignature:    "invalid signature",
	PeerUnknownMessage:      "unknown message code",
	PeerUnmarshalError:      "message unmarshal error",
	PeerNoPermission:        "no permission",
	PeerDifferentGenesis:    "different genesis",
	PeerInvalidBlock:        "invalid block",
	PeerInvalidMessage:      "invalid message",
	PeerBanned:              "banned",
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

func (e PeerError) Serialize() ([]byte, error) {
	return []byte{
		byte(e),
	}, nil
}

type Exception byte

const (
	ExpMissing     Exception = iota // I don`t have the resource you requested
	ExpUnsolicited                  // the request must have pre-checked
	ExpUnauthorized
	ExpServerError
	ExpChunkNotMatch
	ExpOther
)

var exception = map[Exception]string{
	ExpMissing:       "missing resource",
	ExpUnsolicited:   "unsolicited request",
	ExpUnauthorized:  "unauthorized",
	ExpServerError:   "server error",
	ExpOther:         "other exception",
	ExpChunkNotMatch: "chunk not match",
}

func (exp Exception) String() string {
	str, ok := exception[exp]
	if ok {
		return str
	}

	return "unknown exception"
}

func (exp Exception) Error() string {
	return exp.String()
}

func (exp Exception) Serialize() ([]byte, error) {
	return []byte{byte(exp)}, nil
}

func (exp *Exception) Deserialize(buf []byte) error {
	*exp = Exception(buf[0])
	return nil
}
