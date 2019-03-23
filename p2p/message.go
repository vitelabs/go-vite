package p2p

import (
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/p2p/protos"
)

type ProtocolID = byte
type Code = byte
type MsgId = uint32

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
	Serialize() ([]byte, error)
}

type Error struct {
	Code    uint32
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Serialize() ([]byte, error) {
	pb := new(protos.Error)
	pb.Code = e.Code
	pb.Message = e.Message

	return proto.Marshal(pb)
}

func (e *Error) Deserialize(data []byte) (err error) {
	pb := new(protos.Error)
	err = proto.Unmarshal(data, pb)
	if err != nil {
		return
	}

	e.Code = pb.Code
	e.Message = pb.Message

	return nil
}
