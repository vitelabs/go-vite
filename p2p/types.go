package p2p

import (
	"github.com/golang/protobuf/proto"
	"github.com/seiflotfy/cuckoofilter"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"github.com/vitelabs/go-vite/p2p/protos"
	"io"
	"io/ioutil"
	"strconv"
	"time"
)

// @section NetworkID
type NetworkID uint64

const (
	MainNet NetworkID = iota + 1
	Aquarius
	Pisces
	Aries
	Taurus
	Gemini
	Cancer
	Leo
	Virgo
	Libra
	Scorpio
	Sagittarius
	Capricorn
)

var network = [...]string{
	MainNet:     "MainNet",
	Aquarius:    "Aquarius",
	Pisces:      "Pisces",
	Aries:       "Aries",
	Taurus:      "Taurus",
	Gemini:      "Gemini",
	Cancer:      "Cancer",
	Leo:         "Leo",
	Virgo:       "Virgo",
	Libra:       "Libra",
	Scorpio:     "Scorpio",
	Sagittarius: "Sagittarius",
	Capricorn:   "Capricorn",
}

func (i NetworkID) String() string {
	if i >= MainNet && i <= Capricorn {
		return network[i]
	}

	return "Unknown"
}

// @section connFlag
type connFlag int

const (
	outbound connFlag = 1 << iota
	inbound
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
	CmdSetID uint64
	Cmd      uint64
	// how many bytes in payload, used to quickly determine whether payload is valid
	Size       uint64
	Payload    io.Reader
	ReceivedAt time.Time
}

func (msg *Msg) Discard() error {
	_, err := io.Copy(ioutil.Discard, msg.Payload)
	return err
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
	Deserialize(buf []byte) error
}

// @section protocol
type Protocol struct {
	// description of the protocol
	Name string
	// use for message command set, should be unique
	ID uint64
	// read and write Msg with rw
	Handle func(p *Peer, rw MsgReadWriter) error
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
	NetID NetworkID
	// peer id
	ID discovery.NodeID
	// command set supported
	CmdSets []*CmdSet
}

func (hs *Handshake) Serialize() ([]byte, error) {
	cmdsets := make([]*protos.CmdSet, len(hs.CmdSets))

	for i, cmdset := range hs.CmdSets {
		cmdsets[i] = cmdset.Proto()
	}

	hspb := &protos.Handshake{
		NetID:   uint64(hs.NetID),
		Name:    hs.Name,
		ID:      hs.ID[:],
		CmdSets: cmdsets,
	}

	return proto.Marshal(hspb)
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
	hs.NetID = NetworkID(pb.NetID)
	hs.Name = pb.Name

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

// @section topo
type Topo struct {
	Pivot string
	Peers []string
}

func (t *Topo) Serialize() ([]byte, error) {
	return proto.Marshal(&protos.Topo{
		Pivot: t.Pivot,
		Peers: t.Peers,
	})
}

func (t *Topo) Deserialize(buf []byte) error {
	pb := new(protos.Topo)
	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	t.Pivot = pb.Pivot
	t.Peers = pb.Peers

	return nil
}

type topoHandler struct {
	record cuckoofilter.CuckooFilter
}

func (th *topoHandler) Add(hash types.Hash) {

	th.record.Insert(hash[:])
}
func (th *topoHandler) Has(hash types.Hash) bool {
	return th.record.Lookup(hash[:])
}

func (th *topoHandler) Broadcast(topo *Topo, peers []*Peer) {
	for _, peer := range peers {
		go Send(peer.rw, baseProtocolCmdSet, topoCmd, topo)
	}
}
