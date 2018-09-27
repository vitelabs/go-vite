package p2p

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"strconv"
)

// @section peer error
type DiscReason uint64

const (
	DiscRequested DiscReason = iota
	DiscNetworkError
	DiscProtocolError
	DiscUselessPeer
	DiscTooManyPeers
	DiscTooManyPassivePeers
	DiscAlreadyConnected
	DiscIncompatibleVersion
	DiscInvalidIdentity
	DiscQuitting
	DiscUnexpectedIdentity
	DiscSelf
	DiscReadTimeout
	DiscResponseTimeout
	DiscSubprotocolError = 0x10
)

var discReasonStr = [...]string{
	DiscRequested:           "disconnect requested",
	DiscNetworkError:        "network error",
	DiscProtocolError:       "breach of protocol",
	DiscUselessPeer:         "useless peer",
	DiscTooManyPeers:        "too many peers",
	DiscTooManyPassivePeers: "too many passive peers",
	DiscAlreadyConnected:    "already connected",
	DiscIncompatibleVersion: "incompatible p2p protocol version",
	DiscInvalidIdentity:     "invalid node identity",
	DiscQuitting:            "client quitting",
	DiscUnexpectedIdentity:  "unexpected identity",
	DiscSelf:                "connected to self",
	DiscReadTimeout:         "read timeout",
	DiscResponseTimeout:     "waite response timeout",
	DiscSubprotocolError:    "subprotocol error",
}

func (d DiscReason) String() string {
	r64 := uint64(d)
	if uint64(len(discReasonStr)) < r64 {
		return "unknown disconnect reason " + strconv.FormatUint(r64, 10)
	}

	return discReasonStr[r64]
}

func isValidDiscReason(cmd uint64) bool {
	// todo can be more flexible and robust
	code := DiscReason(cmd)
	return (code >= DiscRequested && code <= DiscReadTimeout) || (code == DiscSubprotocolError)
}

func (d DiscReason) Error() string {
	return d.String()
}

func (d DiscReason) Serialize() ([]byte, error) {
	r64 := uint64(d)
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, r64)
	return buf[:n], nil
}

var errDiscReasonDesc = errors.New("cannot call DiscReason.Deserialize")

// just implement Serializable interface
func (d DiscReason) Deserialize(buf []byte) error {
	panic(errDiscReasonDesc)
}

func DeserializeDiscReason(buf []byte) (DiscReason, error) {
	r64, _ := binary.Uvarint(buf)

	if isValidDiscReason(r64) {
		return DiscReason(r64), nil
	}

	return 0, fmt.Errorf("unknown disconnect reason %d", r64)
}

func ReadDiscReason(r io.Reader) (DiscReason, error) {
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return 0, err
	}
	return DeserializeDiscReason(buf)
}

func errTodiscReason(err error) DiscReason {
	if reason, ok := err.(DiscReason); ok {
		return reason
	}
	return DiscSubprotocolError
}
