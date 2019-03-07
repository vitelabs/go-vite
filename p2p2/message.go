package p2p

import (
	"time"
)

type ProtocolID = byte
type Code = byte

type Msg struct {
	Pid        ProtocolID
	Code       Code
	Id         uint32
	Payload    []byte
	ReceivedAt time.Time
	Sender     Peer
}

type MsgReader interface {
	ReadMsg() (Msg, error)
}

type MsgWriter interface {
	WriteMsg(Msg) error
}

type MsgReadWriter interface {
	MsgReader
	MsgWriter
}

type Serializable interface {
	Serialize() []byte
}
