package p2p

import (
	"github.com/vitelabs/go-vite/p2p/protos"
	"github.com/golang/protobuf/proto"
	"fmt"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/crypto"
	"bytes"
	"errors"
	"net"
	"github.com/vitelabs/go-vite/common/types"
	"log"
)

const version byte = 1
const (
	pingCode byte = iota + 1
	pongCode
	findnodeCode
	neighborsCode
)
// the full packet must be little than 1400bytes, consist of:
// version(1 byte), code(1 byte), checksum(32 bytes), signature(64 bytes), payload
// consider varint encoding of protobuf, 1 byte maybe take up 2 bytes after encode.
// so the payload should be little than 1200 bytes.
const maxPacketLength = 1400
const maxPayloadLength = 1200
const maxNeighborsNodes = 10

var errUnmatchedVersion = errors.New("unmatched version")
var errWrongHash = errors.New("validate packet error: wrong hash")
var errInvalidSig = errors.New("validate packet error: invalid signature")

type Message interface {
	getID() NodeID
	Serialize() ([]byte, error)
	Deserialize([]byte) error
	Pack(ed25519.PrivateKey) ([]byte, types.Hash, error)
	Handle(*discover, *net.UDPAddr, types.Hash) error
}

// message Ping
type Ping struct {
	ID NodeID
}

func (p *Ping) getID() NodeID {
	return p.ID
}

func (p *Ping) Serialize() ([]byte, error) {
	pingpb := &protos.Ping{
		ID: p.ID[:],
	}
	return proto.Marshal(pingpb)
}

func (p *Ping) Deserialize(buf []byte) error {
	pingpb := &protos.Ping{}
	err := proto.Unmarshal(buf, pingpb)
	if err != nil {
		return err
	}
	copy(p.ID[:], pingpb.ID)
	return nil
}

func (p *Ping) Pack(key ed25519.PrivateKey) (data []byte, hash types.Hash, err error) {
	buf, err := p.Serialize()
	if err != nil {
		return nil, hash, err
	}

	data, hash = composePacket(key, pingCode, buf)
	return data, hash, nil
}

func (p *Ping) Handle(d *discover, origin *net.UDPAddr, hash types.Hash) error {
	log.Printf("receive ping from %s\n", origin)

	pong := &Pong{
		ID: d.getID(),
		Ping: hash,
	}

	err := d.send(origin, pongCode, pong)
	if err != nil {
		return fmt.Errorf("send pong to %s error: %v\n", origin, err)
	}

	n := NewNode(p.ID, origin.IP, uint16(origin.Port))
	d.tab.addNode(n)
	return nil
}

// message Pong
type Pong struct {
	ID NodeID
	Ping types.Hash
}

func (p *Pong) getID() NodeID {
	return p.ID
}

func (p *Pong) Serialize() ([]byte, error) {
	pongpb := &protos.Pong{
		ID: p.ID[:],
		Ping: p.Ping[:],
	}
	return proto.Marshal(pongpb)
}

func (p *Pong) Deserialize(buf []byte) error {
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
	buf, err := p.Serialize()
	if err != nil {
		return nil, hash, err
	}

	data, hash = composePacket(key, pongCode, buf)
	return data, hash,nil
}

func (p *Pong) Handle(d *discover, origin *net.UDPAddr, hash types.Hash) error {
	log.Printf("receive pong from %s\n", origin)
	return d.receive(pongCode, p)
}

type FindNode struct {
	ID NodeID
	Target NodeID
}

func (p *FindNode) getID() NodeID {
	return p.ID
}

func (f *FindNode) Serialize() ([]byte, error) {
	findpb := &protos.FindNode{
		ID: f.ID[:],
		Target: f.Target[:],
	}
	return proto.Marshal(findpb)
}

func (f *FindNode) Deserialize(buf []byte) error {
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
	buf, err := p.Serialize()
	if err != nil {
		return nil, hash, err
	}

	data, hash = composePacket(priv, findnodeCode, buf)
	return data, hash,nil
}

func (p *FindNode) Handle(d *discover, origin *net.UDPAddr, hash types.Hash) error {
	log.Printf("receive findnode %s from %s\n", p.Target, origin)

	closet := d.tab.closest(p.Target, K)

	nodes := make([]*Node, 0, maxNeighborsNodes)
	for _, node := range closet.nodes {
		nodes = append(nodes, node)
		if len(nodes) == cap(nodes) {
			break
		}
	}

	err := d.send(origin, neighborsCode, &Neighbors{
		ID: d.getID(),
		Nodes: nodes,
	})

	if err != nil {
		log.Printf("send %d neighbors to %s, target: %s, error: %v\n", len(nodes), origin, p.Target, err)
	} else {
		log.Printf("send %d neighbors to %s, target: %s\n", len(nodes), origin, p.Target)
	}

	return err

	//if count > 0 {
	//	nodes := make([]*Node, 0, maxNeighborsNodes)
	//	m := &Neighbors{
	//		ID: d.getID(),
	//	}
	//	// send closet.nodes several times
	//	for i := 0; i < count; i++ {
	//		nodes = append(nodes, closet.nodes[i])
	//
	//		if len(nodes) == cap(nodes) {
	//			m.Nodes = nodes
	//			err := d.send(origin, neighborsCode, m)
	//			if err != nil {
	//				log.Printf("send %d neighbors to %s error: %v\n", len(nodes), origin, err)
	//			} else {
	//				log.Printf("send %d neighbors to %s\n", len(nodes), origin)
	//				nodes = nodes[:0]
	//			}
	//		}
	//	}
	//	if len(nodes) > 0 {
	//		m.Nodes = nodes
	//
	//		err := d.send(origin, neighborsCode, m)
	//		if err != nil {
	//			return fmt.Errorf("send %d neighbors to %s error: %v\n", len(nodes), origin, err)
	//		} else {
	//			log.Printf("send %d neighbors to %s\n", len(nodes), origin)
	//		}
	//	}
	//} else {
	//	log.Printf("findnode %s got 0 closet nodes", p.Target)
	//}
}

type Neighbors struct {
	ID NodeID
	Nodes []*Node
}

func (p *Neighbors) getID() NodeID {
	return p.ID
}

func (n *Neighbors) Serialize() ([]byte, error) {
	nodepbs := make([]*protos.Node, 0, len(n.Nodes))
	for _, node := range n.Nodes {
		nodepbs = append(nodepbs, node.toProto())
	}

	neighborspb := &protos.Neighbors{
		ID: n.ID[:],
		Nodes: nodepbs,
	}
	return proto.Marshal(neighborspb)
}

func (n *Neighbors) Deserialize(buf []byte) error {
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
	buf, err := p.Serialize()
	if err != nil {
		return nil, hash, err
	}

	data, hash = composePacket(priv, neighborsCode, buf)
	return data, hash,nil
}

func (p *Neighbors) Handle(d *discover, origin *net.UDPAddr, hash types.Hash) error {
	log.Printf("receive %d neighbors from %s\n", len(p.Nodes), p.getID())
	return d.receive(neighborsCode, p)
}

// version code checksum signature payload
func composePacket(priv ed25519.PrivateKey, code byte, payload []byte) (data []byte, hash types.Hash) {
	data = []byte{version, code}

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
		return nil, hash, errUnmatchedVersion
	}

	pktCode := packet[1]
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

func decode(code byte, payload []byte) (m Message, err error) {
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
		return nil, fmt.Errorf("decode packet error: unknown code %d", code)
	}

	m.Deserialize(payload)

	return m, err
}
