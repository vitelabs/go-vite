package p2p

import (
	"net"
	"log"
	"fmt"
	"github.com/vitelabs/go-vite/crypto/ed25519"
)

// @section Msg
type Serializable interface {
	Serialize() ([]byte, error)
}

type Msg struct {
	Code uint64
	Payload Serializable
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
	Close(error)
}

func Send(w MsgWriter, msg *Msg) error {
	return w.WriteMsg(*msg)
}

// transport use protobuf to serialize/deserialize message.
func NewPBTS(priv ed25519.PrivateKey, peerID NodeID, conn net.Conn) transport {
	return &PBTS{
		peerID: peerID,
		priv: priv,
		conn: conn,
	}
}


// @section PBTS
//
const headerLenth = 100
const bodyLenth = 32
const maxBodySize uint = ^(0 << 32)


type PBTS struct {
	peerID NodeID
	priv ed25519.PrivateKey
	conn net.Conn
}

func (pt *PBTS) ReadMsg() (Msg, error) {

}

func (pt *PBTS) WriteMsg(m Msg) error {
	data, err := pack(m)

	if err != nil {
		return fmt.Errorf("pack message error: %v\n", err)
	}

	n, err := pt.conn.Write(data)
	if err != nil {
		return fmt.Errorf("write message error: %v\n", err)
	}
	if n != len(data) {
		return fmt.Errorf("write incomplete message: %d / %d\n", n, len(data))
	}

	return nil
}

func (pt *PBTS) Handshake(our *protoHandshake) (*protoHandshake, error) {

}

func (pt *PBTS) Close(err error) {
	pt.conn.Close()
	log.Printf("pbts %s close, reason: %v\n", pt.peerID, err)
}

func pack(m Msg) ([]byte, error) {

}

func unpackMsg([]byte) (Msg, error) {

}
