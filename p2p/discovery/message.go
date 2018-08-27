package discovery

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/p2p/discovery/protos"
	"net"
)

const version byte = 1

type packetCode byte

const (
	pingCode packetCode = iota
	pongCode
	findnodeCode
	neighborsCode
)

var packetStrs = [...]string{
	"ping",
	"pong",
	"findnode",
	"neighbors",
}

func (c packetCode) String() string {
	return packetStrs[c]
}

// the full packet must be little than 1400bytes, consist of:
// version(1 byte), code(1 byte), checksum(32 bytes), signature(64 bytes), payload
// consider varint encoding of protobuf, 1 byte maybe take up 2 bytes after encode.
// so the payload should be little than 1200 bytes.
const maxPacketLength = 1400

//const maxPayloadLength = 1200
const maxNeighborsNodes = 10

var errUnmatchedVersion = errors.New("unmatched version")
var errWrongHash = errors.New("validate packet error: invalid hash")
var errInvalidSig = errors.New("validate packet error: invalid signature")

type Message interface {
	getID() NodeID
	serialize() ([]byte, error)
	deserialize([]byte) error
	Pack(ed25519.PrivateKey) ([]byte, types.Hash, error)
	Handle(*udpAgent, *net.UDPAddr, types.Hash) error
}

// message Ping
type Ping struct {
	ID NodeID
}

func (p *Ping) getID() NodeID {
	return p.ID
}

func (p *Ping) serialize() ([]byte, error) {
	pingpb := &protos.Ping{
		ID: p.ID[:],
	}
	return proto.Marshal(pingpb)
}

func (p *Ping) deserialize(buf []byte) error {
	pingpb := &protos.Ping{}
	err := proto.Unmarshal(buf, pingpb)
	if err != nil {
		return err
	}
	copy(p.ID[:], pingpb.ID)
	return nil
}

func (p *Ping) Pack(key ed25519.PrivateKey) (data []byte, hash types.Hash, err error) {
	buf, err := p.serialize()
	if err != nil {
		return nil, hash, err
	}

	data, hash = composePacket(key, pingCode, buf)
	return data, hash, nil
}

func (p *Ping) Handle(d *udpAgent, origin *net.UDPAddr, hash types.Hash) error {
	discvLog.Info("receive ping ", "from", origin)

	pong := &Pong{
		ID:   d.ID(),
		Ping: hash,
	}

	err := d.send(origin, pongCode, pong)
	if err != nil {
		return fmt.Errorf("send pong to %s error: %v\n", origin, err)
	}

	n := newNode(p.ID, origin.IP, uint16(origin.Port))
	d.tab.addNode(n)
	return nil
}

// message Pong
type Pong struct {
	ID   NodeID
	Ping types.Hash
}

func (p *Pong) getID() NodeID {
	return p.ID
}

func (p *Pong) serialize() ([]byte, error) {
	pongpb := &protos.Pong{
		ID:   p.ID[:],
		Ping: p.Ping[:],
	}
	return proto.Marshal(pongpb)
}

func (p *Pong) deserialize(buf []byte) error {
	pongpb := &protos.Pong{}
	err := proto.Unmarshal(buf, pongpb)
	if err != nil {
		return err
	}
	copy(p.ID[:], pongpb.ID)
	copy(p.Ping[:], pongpb.Ping)
	return nil
}

func (p *Pong) Pack(key ed25519.PrivateKey) (data []byte, hash types.Hash, err error) {
	buf, err := p.serialize()
	if err != nil {
		return nil, hash, err
	}

	data, hash = composePacket(key, pongCode, buf)
	return data, hash, nil
}

func (p *Pong) Handle(d *udpAgent, origin *net.UDPAddr, hash types.Hash) error {
	discvLog.Info(fmt.Sprintf("receive pong from %s\n", origin))
	return d.receive(pongCode, p)
}

type FindNode struct {
	ID     NodeID
	Target NodeID
}

func (p *FindNode) getID() NodeID {
	return p.ID
}

func (f *FindNode) serialize() ([]byte, error) {
	findpb := &protos.FindNode{
		ID:     f.ID[:],
		Target: f.Target[:],
	}
	return proto.Marshal(findpb)
}

func (f *FindNode) deserialize(buf []byte) error {
	findpb := &protos.FindNode{}
	err := proto.Unmarshal(buf, findpb)
	if err != nil {
		return err
	}

	copy(f.ID[:], findpb.ID)
	copy(f.Target[:], findpb.Target)
	return nil
}

func (p *FindNode) Pack(priv ed25519.PrivateKey) (data []byte, hash types.Hash, err error) {
	buf, err := p.serialize()
	if err != nil {
		return nil, hash, err
	}

	data, hash = composePacket(priv, findnodeCode, buf)
	return data, hash, nil
}

func (p *FindNode) Handle(d *udpAgent, origin *net.UDPAddr, hash types.Hash) error {
	discvLog.Info(fmt.Sprintf("receive findnode %s from %s\n", p.Target, origin))

	closet := d.tab.closest(p.Target, maxNeighborsNodes)

	err := d.send(origin, neighborsCode, &Neighbors{
		ID:    d.ID(),
		Nodes: closet.nodes,
	})

	if err != nil {
		discvLog.Info(fmt.Sprintf("send %d neighbors to %s, target: %s, error: %v\n", len(closet.nodes), origin, p.Target, err))
	} else {
		discvLog.Info(fmt.Sprintf("send %d neighbors to %s, target: %s\n", len(closet.nodes), origin, p.Target))
	}

	return err
}

type Neighbors struct {
	ID    NodeID
	Nodes []*Node
}

func (p *Neighbors) getID() NodeID {
	return p.ID
}

func (n *Neighbors) serialize() ([]byte, error) {
	nodepbs := make([]*protos.Node, 0, len(n.Nodes))
	for _, node := range n.Nodes {
		nodepbs = append(nodepbs, node.toProto())
	}

	neighborspb := &protos.Neighbors{
		ID:    n.ID[:],
		Nodes: nodepbs,
	}
	return proto.Marshal(neighborspb)
}

func (n *Neighbors) deserialize(buf []byte) error {
	neighborspb := &protos.Neighbors{}
	err := proto.Unmarshal(buf, neighborspb)
	if err != nil {
		return err
	}

	copy(n.ID[:], neighborspb.ID)

	nodes := make([]*Node, 0, len(neighborspb.Nodes))

	for _, nodepb := range neighborspb.Nodes {
		nodes = append(nodes, protoToNode(nodepb))
	}

	n.Nodes = nodes

	return nil
}

func (p *Neighbors) Pack(priv ed25519.PrivateKey) (data []byte, hash types.Hash, err error) {
	buf, err := p.serialize()
	if err != nil {
		return nil, hash, err
	}

	data, hash = composePacket(priv, neighborsCode, buf)
	return data, hash, nil
}

func (p *Neighbors) Handle(d *udpAgent, origin *net.UDPAddr, hash types.Hash) error {
	discvLog.Info(fmt.Sprintf("receive %d neighbors from %s\n", len(p.Nodes), p.getID()))
	return d.receive(neighborsCode, p)
}

// version code checksum signature payload
func composePacket(priv ed25519.PrivateKey, code packetCode, payload []byte) (data []byte, hash types.Hash) {
	data = []byte{version, byte(code)}

	sig := ed25519.Sign(priv, payload)
	checksum := crypto.Hash(32, append(sig, payload...))

	data = append(data, checksum...)
	data = append(data, sig...)
	data = append(data, payload...)

	copy(hash[:], checksum)
	return data, hash
}

func unPacket(packet []byte) (m Message, hash types.Hash, err error) {
	pktVersion := packet[0]

	if pktVersion != version {
		return nil, hash, fmt.Errorf("unmatched version: %d / %d\n", pktVersion, version)
	}

	pktCode := packetCode(packet[1])
	pktHash := packet[2:34]
	payloadWithSig := packet[34:]
	pktSig := packet[34:98]
	payload := packet[98:]

	// compare checksum
	reHash := crypto.Hash(32, payloadWithSig)
	if !bytes.Equal(reHash, pktHash) {
		return nil, hash, errWrongHash
	}

	// unpack packet to get content and signature
	m, err = decode(pktCode, payload)
	if err != nil {
		return nil, hash, err
	}

	// verify signature
	id := m.getID()
	pub := id[:]
	valid, err := crypto.VerifySig(pub, payload, pktSig)
	if err != nil {
		return nil, hash, err
	}
	if valid {
		copy(hash[:], pktHash)
		return m, hash, nil
	}

	return nil, hash, errInvalidSig
}

func decode(code packetCode, payload []byte) (m Message, err error) {
	switch code {
	case pingCode:
		m = new(Ping)
	case pongCode:
		m = new(Pong)
	case findnodeCode:
		m = new(FindNode)
	case neighborsCode:
		m = new(Neighbors)
	default:
		return m, fmt.Errorf("decode packet error: unknown code %d", code)
	}

	err = m.deserialize(payload)

	return m, err
}
