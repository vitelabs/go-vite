package p2p

import (
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p/protos"
	"net"
)

type NetworkID uint32

const Version uint32 = 1

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
	Code    uint64
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
	NetID   NetworkID
	Name    string
	ID      NodeID
	Version uint32
}

func (hs *Handshake) Serialize() ([]byte, error) {
	hspb := &protos.Handshake{
		NetID:   uint32(hs.NetID),
		Name:    hs.Name,
		ID:      hs.ID[:],
		Version: hs.Version,
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
	hs.Version = hspb.Version
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
		log:  log15.New("module", "p2p/transport"),
	}
}

// @section PBTS
const headerLength = 20
const maxPayloadSize = ^uint32(0) >> 8

type PBTS struct {
	//peerID NodeID
	//priv ed25519.PrivateKey
	conn net.Conn
	log  log15.Logger
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
	size := binary.BigEndian.Uint32(header[8:12])

	if size > maxPayloadSize {
		return m, fmt.Errorf("msg %d payload too large: %d / %d\n", m.Code, size, maxPayloadSize)
	}

	// read payload according to size
	if size > 0 {
		payload := make([]byte, size)

		err = readFullBytes(pt.conn, payload)
		if err != nil {
			return m, fmt.Errorf("read msg %d payload (%d bytes) error: %v\n", m.Code, size, err)
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
	data, err := pack(m)

	if err != nil {
		return fmt.Errorf("pack smg %d (%d bytes) to %s error: %v\n", m.Code, len(m.Payload), pt.conn.RemoteAddr(), err)
	}

	n, err := pt.conn.Write(data)
	if err != nil {
		return fmt.Errorf("write msg %d (%d bytes) to %s error: %v\n", m.Code, len(m.Payload), pt.conn.RemoteAddr(), err)
	}
	if n != len(data) {
		return fmt.Errorf("write incomplete msg to %s: %d / %d\n", pt.conn.RemoteAddr(), n, len(data))
	}

	return nil
}

func (pt *PBTS) Handshake(our *Handshake) (*Handshake, error) {
	data, err := our.Serialize()
	if err != nil {
		return nil, fmt.Errorf("our handshake with %s serialize error: %v\n", pt.conn.RemoteAddr(), err)
	}

	sendErr := make(chan error, 1)
	go func() {
		sendErr <- pt.WriteMsg(Msg{
			Code:    handshakeMsg,
			Payload: data,
		})
	}()

	msg, err := pt.ReadMsg()
	if err != nil {
		<-sendErr
		return nil, fmt.Errorf("read handshake from %s error: %v\n", pt.conn.RemoteAddr(), err)
	}

	if msg.Code != handshakeMsg {
		return nil, fmt.Errorf("need handshake from %s, got %d\n", pt.conn.RemoteAddr(), msg.Code)
	}

	hs := &Handshake{}
	err = hs.Deserialize(msg.Payload)
	if err != nil {
		return nil, fmt.Errorf("handshake from %s deserialize error: %v\n", pt.conn.RemoteAddr(), err)
	}

	if hs.Version != our.Version {
		return nil, fmt.Errorf("unmatched version\n")
	}

	if hs.NetID != our.NetID {
		return nil, fmt.Errorf("unmatched network id: %d / %d from %s\n", hs.NetID, our.NetID, pt.conn.RemoteAddr())
	}

	if err := <-sendErr; err != nil {
		return nil, fmt.Errorf("send handshake to %s error: %v\n", pt.conn.RemoteAddr(), err)
	}

	return hs, nil
}

func (pt *PBTS) Close(err error) {
	reason := errTodiscReason(err)
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, uint64(reason))

	err = pt.WriteMsg(Msg{
		Code:    discMsg,
		Payload: data,
	})
	if err != nil {
		pt.log.Error("send disc msg error", "to", pt.conn.RemoteAddr().String(), "error", err)
	}

	pt.conn.Close()
	pt.log.Info("disconnect", "to", pt.conn.RemoteAddr().String(), "error", err)
}

func pack(m Msg) (data []byte, err error) {
	if uint32(len(m.Payload)) > maxPayloadSize {
		return data, fmt.Errorf("msg %d payload too large: %d / %d\n", m.Code, len(m.Payload), maxPayloadSize)
	}

	header := make([]byte, headerLength)

	// add code to header
	binary.BigEndian.PutUint64(header[:8], m.Code)

	// sign payload length to header
	size := uint32(len(m.Payload))
	binary.BigEndian.PutUint32(header[8:12], size)

	// concat header and payload
	if size == 0 {
		return header, nil
	}

	data = append(header, m.Payload...)
	return data, nil
}
