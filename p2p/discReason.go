package p2p

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
)

// @section peer error
type DiscReason uint64

const (
	DiscRequested DiscReason = iota + 1
	DiscNetworkError
	DiscAllProtocolDone
	DiscProtocolError
	DiscUselessPeer
	DiscTooManyPeers
	DiscTooManyInboundPeers
	DiscAlreadyConnected
	DiscIncompatibleVersion
	DiscInvalidIdentity
	DiscQuitting
	DiscSelf
	DiscReadTimeout
	DiscReadError
	DiscResponseTimeout
	DiscUnKnownProtocol
	DiscHandshakeFail
)

var discReasonStr = [...]string{
	DiscRequested:           "disconnect requested",
	DiscNetworkError:        "network error",
	DiscAllProtocolDone:     "all protocols done",
	DiscProtocolError:       "protocol error",
	DiscUselessPeer:         "useless peer",
	DiscTooManyPeers:        "too many peers",
	DiscTooManyInboundPeers: "too many passive peers",
	DiscAlreadyConnected:    "already connected",
	DiscIncompatibleVersion: "incompatible p2p version",
	DiscInvalidIdentity:     "invalid node identity",
	DiscQuitting:            "client quitting",
	DiscSelf:                "connected to self",
	DiscReadTimeout:         "read timeout",
	DiscReadError:           "read error",
	DiscResponseTimeout:     "wait response timeout",
	DiscUnKnownProtocol:     "missing protocol handler",
	DiscHandshakeFail:       "p2p handshake error",
}

func (d DiscReason) String() string {
	if d > DiscHandshakeFail {
		return "unknown disc reason"
	}
	return discReasonStr[d]
}

func (d DiscReason) Error() string {
	return d.String()
}

func (d DiscReason) Serialize() ([]byte, error) {
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, uint64(d))
	return buf[:n], nil
}

var errDiscReasonDesc = errors.New("can`t call DiscReason.Deserialize")

// just implement Serializable interface
func (d DiscReason) Deserialize(buf []byte) error {
	panic(errDiscReasonDesc)
}

var errIncompleteReason = errors.New("incomplete reason bytes")

func isValidDiscReason(reason uint64) bool {
	return reason <= uint64(len(discReasonStr))
}

func DeserializeDiscReason(buf []byte) (reason DiscReason, err error) {
	r64, n := binary.Uvarint(buf)

	if n != len(buf) {
		err = errIncompleteReason
		return
	}

	if isValidDiscReason(r64) {
		reason = DiscReason(r64)
		return
	}

	err = fmt.Errorf("unknown disconnect reason %d", r64)
	return
}
