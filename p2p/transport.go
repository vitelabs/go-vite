package p2p

import "net"

type Msg struct {
	Code       uint64
	Payload    []byte
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

type protoHandshake struct {
	Name    string
	ID      NodeID
}

type transport interface {
	Handshake(our *protoHandshake) (*protoHandshake, error)
	MsgReadWriter
	Close(err error)
}

func Send(w MsgWriter, msg *Msg) error {
// todo
	return nil
}

func protobufTS(conn net.Conn) transport {

}
