package p2p

import (
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"github.com/vitelabs/go-vite/p2p/network"
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

type CmdSet struct {
	ID   uint64
	Name string
}

func (s *CmdSet) Serialize() ([]byte, error) {
	return proto.Marshal(s.Proto())
}

func (s *CmdSet) Deserialize(buf []byte) error {
	pb := new(protos.CmdSet)
	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}
	s.ID = pb.ID
	s.Name = pb.Name
	return nil
}

func (s *CmdSet) String() string {
	return s.Name + "/" + strconv.FormatUint(s.ID, 10)
}

func (s *CmdSet) Proto() *protos.CmdSet {
	return &protos.CmdSet{
		ID:   s.ID,
		Name: s.Name,
	}
}

// @section Msg
type Msg struct {
	CmdSet     uint64
	Cmd        uint32
	Id         uint64 // as message context
	Size       uint32 // how many bytes in payload, used to quickly determine whether payload is valid
	Payload    []byte
	ReceivedAt time.Time
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
	ID uint64
	// read and write Msg with rw
	Handle func(p *Peer, rw *ProtoFrame) error
}

func (p *Protocol) String() string {
	return p.Name + "/" + strconv.FormatUint(p.ID, 10)
}

func (p *Protocol) CmdSet() *CmdSet {
	return &CmdSet{
		ID:   p.ID,
		Name: p.Name,
	}
}

// handshake message
type Handshake struct {
	Version uint64
	// peer name, use for readability and log
	Name string
	// running at which network
	NetID network.ID
	// peer remoteID
	ID discovery.NodeID
	// command set supported
	CmdSets []*CmdSet
	// peer`s IP
	RemoteIP net.IP
	// peer`s Port
	RemotePort uint16
}

func (hs *Handshake) Serialize() ([]byte, error) {
	cmdsets := make([]*protos.CmdSet, len(hs.CmdSets))

	for i, cmdset := range hs.CmdSets {
		cmdsets[i] = cmdset.Proto()
	}

	return proto.Marshal(&protos.Handshake{
		NetID:      uint64(hs.NetID),
		Name:       hs.Name,
		ID:         hs.ID[:],
		CmdSets:    cmdsets,
		RemoteIP:   hs.RemoteIP,
		RemotePort: uint32(hs.RemotePort),
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

	hs.Version = pb.Version
	hs.ID = id
	hs.NetID = network.ID(pb.NetID)
	hs.Name = pb.Name
	hs.RemoteIP = pb.RemoteIP
	hs.RemotePort = uint16(pb.RemotePort)

	cmdsets := make([]*CmdSet, len(pb.CmdSets))
	for i, cmdset := range pb.CmdSets {
		cmdsets[i] = &CmdSet{
			ID:   cmdset.ID,
			Name: cmdset.Name,
		}
	}

	hs.CmdSets = cmdsets

	return nil
}
