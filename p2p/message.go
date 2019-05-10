package p2p

import (
	"time"

	"github.com/vitelabs/go-vite/tools/bytes_pool"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/p2p/protos"
)

type Code = byte
type MsgId = uint32

type Msg struct {
	Code       Code
	Id         uint32
	Payload    []byte
	ReceivedAt time.Time
	Sender     Peer
}

// Recycle will put Msg.Payload back to pool
func (m Msg) Recycle() {
	bytes_pool.Put(m.Payload)
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

func Disconnect(w MsgWriter, err error) (e2 error) {
	var msg = Msg{
		Code: baseDisconnect,
	}

	if err != nil {
		var e *Error
		if pe, ok := err.(PeerError); ok {
			e = &Error{
				Code:    uint32(pe),
				Message: pe.Error(),
			}
		} else if e, ok = err.(*Error); ok {
			// do nothing
		} else {
			e = &Error{
				Message: err.Error(),
			}
		}

		msg.Payload, e2 = e.Serialize()
		if e2 != nil {
			return e2
		}

		return
	}

	e2 = w.WriteMsg(msg)

	return nil
}
