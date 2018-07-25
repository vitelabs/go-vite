package p2p

import (
	"net"
	"log"
	"fmt"
	"encoding/binary"
	"github.com/vitelabs/go-vite/p2p/protos"

	"github.com/golang/protobuf/proto"
)

type NetworkID uint32

const (
	MainNet NetworkID = iota + 1
	TestNet
)

func (i NetworkID) String() string {
	switch i {
	case MainNet:
		return "MainNet"
	case TestNet:
		return "TestNet"
	default:
		return "Unknown"
	}
}

// @section Msg
type Msg struct {
	Code uint64
	Payload []byte
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

// handshake message
type Handshake struct {
	NetID	NetworkID
	Name    string
	ID      NodeID
}

func (hs *Handshake) Serialize() ([]byte, error) {
	hspb := &protos.Handshake{
		NetID: uint32(hs.NetID),
		Name: hs.Name,
		ID: hs.ID[:],
	}

	return proto.Marshal(hspb)
}

func (hs *Handshake) Deserialize(buf []byte) error {
	hspb := &protos.Handshake{}
	err := proto.Unmarshal(buf, hspb)
	if err != nil {
		return err
	}

	copy(hs.ID[:], hspb.ID)
	hs.Name = hspb.Name
	hs.NetID = NetworkID(hspb.NetID)
	return nil
}

// disc message
type DiscMsg struct {
	reason DiscReason
}

func (d *DiscMsg) Serialize() ([]byte, error) {
	discpb := &protos.Disc{
		Reason: uint32(d.reason),
	}

	return proto.Marshal(discpb)
}

func (d *DiscMsg) Deserialize(buf []byte) error {
	discpb := &protos.Disc{}
	err := proto.Unmarshal(buf, discpb)
	if err != nil {
		return err
	}

	d.reason = DiscReason(discpb.Reason)
	return nil
}


// mean the msg transport link of peers.
type transport interface {
	Handshake(our *Handshake) (*Handshake, error)
	MsgReadWriter
	Close(error)
}

func Send(w MsgWriter, msg *Msg) error {
	return w.WriteMsg(*msg)
}

// transport use protobuf to serialize/deserialize message.
func NewPBTS(conn net.Conn) transport {
	return &PBTS{
		conn: conn,
	}
}

// @section PBTS
const headerLength = 20
const maxPayloadSize = ^uint64(0)

type PBTS struct {
	//peerID NodeID
	//priv ed25519.PrivateKey
	conn net.Conn
}

func (pt *PBTS) ReadMsg() (m Msg, err error) {
	header := make([]byte, headerLength)

	err = readFullBytes(pt.conn, header)
	if err != nil {
		return m, fmt.Errorf("read msg header error: %v\n", err)
	}

	// extract msg.Code
	m.Code = binary.BigEndian.Uint64(header[:8])

	// extract length of payload
	size := binary.BigEndian.Uint64(header[8:16])
	log.Printf("Msg payload length: %d bytes\n", size)
	// read payload according to size
	if size > 0 {
		payload := make([]byte, size)

		err = readFullBytes(pt.conn, payload)
		if err != nil {
			return m, fmt.Errorf("read msg payload error: %v\n", err)
		}

		m.Payload = payload
	}

	return m, nil
}

func readFullBytes(conn net.Conn, data []byte) error {
	length := cap(data)
	index := 0
	for {
		n, err := conn.Read(data[index:])
		if err != nil {
			return err
		}

		index += n
		if index == length {
			break
		}
	}
	return nil
}

func (pt *PBTS) WriteMsg(m Msg) error {
	if uint64(len(m.Payload)) > maxPayloadSize {
		return fmt.Errorf("too large msg payload: %d / %d\n", len(m.Payload), maxPayloadSize)
	}
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

func (pt *PBTS) Handshake(our *Handshake) (*Handshake, error) {
	data, err := our.Serialize()
	if err != nil {
		return nil, fmt.Errorf("handshake serialize error: %v\n", err)
	}

	sendErr := make(chan error, 1)
	go func() {
		sendErr <- pt.WriteMsg(Msg{
			Code: handshakeMsg,
			Payload: data,
		})
	}()

	msg, err := pt.ReadMsg()
	if err != nil {
		<- sendErr
		return nil, fmt.Errorf("read handshake msg error: %v\n", err)
	}

	if msg.Code != handshakeMsg {
		return nil, fmt.Errorf("need handshake msg, got %d\n", msg.Code)
	}

	hs := &Handshake{}
	err = hs.Deserialize(msg.Payload)
	if err != nil {
		return nil, fmt.Errorf("handshake deserialize error: %v\n", err)
	}

	if hs.NetID != our.NetID {
		return nil, fmt.Errorf("unmatched network id: %d / %d\n", hs.NetID, our.NetID)
	}

	if err := <- sendErr; err != nil {
		return nil, fmt.Errorf("send handshake error: %v\n", err)
	}

	return hs, nil
}

func (pt *PBTS) Close(err error) {
	reason := errTodiscReason(err)
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, uint64(reason))

	err = pt.WriteMsg(Msg{
		Code: discMsg,
		Payload: data,
	})
	if err != nil {
		log.Printf("send disc msg error: %v\n", err)
	}

	pt.conn.Close()
	log.Printf("pbts close, reason: %v\n", err)
}

func pack(m Msg) ([]byte, error) {
	header := make([]byte, headerLength)

	// add code to header
	binary.BigEndian.PutUint64(header[:8], m.Code)

	// sign payload length to header
	var size uint64
	if m.Payload == nil {
		size = 0
	} else {
		size = uint64(len(m.Payload))
	}
	binary.BigEndian.PutUint64(header[8:16], size)

	// concat header and payload
	data := make([]byte, headerLength + size)
	copy(data, header)
	if m.Payload != nil {
		copy(data[headerLength:], m.Payload)
	}

	return data, nil
}
