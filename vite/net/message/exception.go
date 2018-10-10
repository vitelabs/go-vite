package message

import (
	"encoding/binary"
	"errors"
)

type Exception uint64

const (
	Missing             Exception = iota // I don`t have the resource you requested
	Canceled                             // the request have been canceled
	Unsolicited                          // the request must have pre-checked
	Blocked                              // you have been blocked
	RepetitiveHandshake                  // handshake should happen only once, as the first msg
	Connected                            // you have been connected with me
	DifferentNet
	UnMatchedMsgVersion
	UnIdenticalGenesis
	FileTransDone
)

var exception = [...]string{
	Missing:             "I don`t have the resource you requested",
	Canceled:            "the request have been canceled",
	Unsolicited:         "your request must have pre-checked",
	Blocked:             "you have been blocked",
	RepetitiveHandshake: "handshake should happen only once, as the first msg",
	Connected:           "you have connected to me",
	DifferentNet:        "we are at different network",
	UnMatchedMsgVersion: "UnMatchedMsgVersion",
	UnIdenticalGenesis:  "UnIdenticalGenesis",
	FileTransDone:       "FileTransDone",
}

func (exp Exception) String() string {
	return exception[exp]
}

func (exp Exception) Error() string {
	return exception[exp]
}

func (exp Exception) Serialize() ([]byte, error) {
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, uint64(exp))
	return buf[:n], nil
}

func (exp Exception) Deserialize(buf []byte) error {
	panic("use deserializeException instead")
}

func DeserializeException(buf []byte) (e Exception, err error) {
	u64, n := binary.Varint(buf)
	if n != len(buf) {
		err = errors.New("use incomplete data")
		return
	}

	return Exception(u64), nil
}
