package p2p

import (
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"github.com/vitelabs/go-vite/p2p/protos"
	"net"
	"strconv"
	"time"
)

// @section connFlag
type connFlag int

const (
	outbound connFlag = 1 << iota
	inbound
	static
)

func (f connFlag) is(f2 connFlag) bool {
	return (f & f2) != 0
}

// @section CmdSetID
type CmdSet = uint32
type Cmd = uint16

// @section Msg
type Msg struct {
	CmdSet     CmdSet
	Cmd        Cmd
	Id         uint64 // as message context
	Payload    []byte
	ReceivedAt time.Time
	SendAt     time.Time
}

func (msg *Msg) Recycle() {
	//msg.CmdSet = 0
	//msg.Cmd = 0
	//msg.Size = 0
	//msg.Payload = nil
	//msg.Id = 0
	//
	//msgPool.Put(msg)
}

type MsgReader interface {
	ReadMsg() (*Msg, error)
}

type MsgWriter interface {
	WriteMsg(*Msg) error
}

type MsgReadWriter interface {
	MsgReader
	MsgWriter
}

type Serializable interface {
	Serialize() ([]byte, error)
	Deserialize(buf []byte) error
}

// @section protocol
type Protocol struct {
	// description of the protocol
	Name string
	// use for message command set, should be unique
	ID CmdSet
	// read and write Msg with rw
	Handle func(p *Peer, rw *ProtoFrame) error
}

func (p *Protocol) String() string {
	return p.Name + "/" + strconv.FormatUint(uint64(p.ID), 10)
}

// handshake message
type Handshake struct {
	// peer name, use for readability and log
	Name string
	// peer remoteID
	ID discovery.NodeID
	// command set supported
	CmdSets []CmdSet
	// peer`s IP
	RemoteIP net.IP
	// peer`s Port
	RemotePort uint16
	// tell peer our tcp listen port, could use for update discovery info
	Port uint16
}

func (hs *Handshake) Serialize() ([]byte, error) {
	return proto.Marshal(&protos.Handshake{
		Name:       hs.Name,
		ID:         hs.ID[:],
		CmdSets:    hs.CmdSets,
		RemoteIP:   hs.RemoteIP,
		RemotePort: uint32(hs.RemotePort),
		Port:       uint32(hs.Port),
	})
}

func (hs *Handshake) Deserialize(buf []byte) error {
	pb := new(protos.Handshake)
	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	id, err := discovery.Bytes2NodeID(pb.ID)
	if err != nil {
		return err
	}

	hs.ID = id
	hs.Name = pb.Name
	hs.RemoteIP = pb.RemoteIP
	hs.RemotePort = uint16(pb.RemotePort)
	hs.CmdSets = pb.CmdSets
	hs.Port = uint16(pb.Port)

	return nil
}

type Transport interface {
	MsgReadWriter
	Close()
	Handshake(our *Handshake) (their *Handshake, err error)
}
