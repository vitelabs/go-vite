package p2p

import (
	"time"

	"github.com/vitelabs/go-vite/tools/bytes_pool"
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

func Disconnect(w MsgWriter, err error) (e2 error) {
	var msg = Msg{
		Code: CodeDisconnect,
	}

	if err != nil {
		if pe, ok := err.(PeerError); ok {
			msg.Payload, _ = pe.Serialize()
		}
	} else {
		msg.Payload, _ = PeerQuitting.Serialize()
	}

	e2 = w.WriteMsg(msg)

	return nil
}
