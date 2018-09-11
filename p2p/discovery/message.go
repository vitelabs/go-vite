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
	"time"
)

const version byte = 1

var expiration = 20 * time.Second

func getExpiration() time.Time {
	return time.Now().Add(expiration)
}

func isExpired(t time.Time) bool {
	return t.Before(time.Now())
}

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
const maxPacketLength = 1200
const maxNeighborsNodes = 10

var errUnmatchedVersion = errors.New("unmatched version")
var errWrongHash = errors.New("validate packet error: invalid hash")
var errInvalidSig = errors.New("validate packet error: invalid signature")

type Message interface {
	serialize() ([]byte, error)
	deserialize([]byte) error
	pack(ed25519.PrivateKey) ([]byte, types.Hash, error)
	isExpired() bool
	sender() NodeID
}

// message Ping
type Ping struct {
	ID         NodeID
	IP         net.IP
	UDP        uint16
	TCP        uint16
	Expiration time.Time
}

func (p *Ping) sender() NodeID {
	return p.ID
}

func (p *Ping) serialize() ([]byte, error) {
	pingpb := &protos.Ping{
		ID:         p.ID[:],
		IP:         p.IP,
		UDP:        uint32(p.UDP),
		TCP:        uint32(p.TCP),
		Expiration: p.Expiration.Unix(),
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
	p.IP = pingpb.IP
	p.UDP = uint16(pingpb.UDP)
	p.TCP = uint16(pingpb.TCP)
	p.Expiration = time.Unix(pingpb.Expiration, 0)

	return nil
}

func (p *Ping) pack(key ed25519.PrivateKey) (data []byte, hash types.Hash, err error) {
	buf, err := p.serialize()
	if err != nil {
		return nil, hash, err
	}

	data, hash = composePacket(key, pingCode, buf)
	return data, hash, nil
}

func (p *Ping) isExpired() bool {
	return isExpired(p.Expiration)
}

// message Pong
type Pong struct {
	ID         NodeID
	Ping       types.Hash
	Expiration time.Time
}

func (p *Pong) sender() NodeID {
	return p.ID
}

func (p *Pong) serialize() ([]byte, error) {
	pongpb := &protos.Pong{
		ID:         p.ID[:],
		Expiration: p.Expiration.Unix(),
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
	p.Expiration = time.Unix(pongpb.Expiration, 0)

	return nil
}

func (p *Pong) pack(key ed25519.PrivateKey) (data []byte, hash types.Hash, err error) {
	buf, err := p.serialize()
	if err != nil {
		return nil, hash, err
	}

	data, hash = composePacket(key, pongCode, buf)
	return data, hash, nil
}

func (p *Pong) isExpired() bool {
	return isExpired(p.Expiration)
}

// @message findnode
type FindNode struct {
	ID         NodeID
	Target     NodeID
	Expiration time.Time
}

func (p *FindNode) sender() NodeID {
	return p.ID
}

func (f *FindNode) serialize() ([]byte, error) {
	findpb := &protos.FindNode{
		ID:         f.ID[:],
		Target:     f.Target[:],
		Expiration: f.Expiration.Unix(),
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
	f.Expiration = time.Unix(findpb.Expiration, 0)

	return nil
}

func (p *FindNode) pack(priv ed25519.PrivateKey) (data []byte, hash types.Hash, err error) {
	buf, err := p.serialize()
	if err != nil {
		return nil, hash, err
	}

	data, hash = composePacket(priv, findnodeCode, buf)
	return data, hash, nil
}

func (p *FindNode) isExpired() bool {
	return isExpired(p.Expiration)
}

// @message neighbors
type Neighbors struct {
	ID         NodeID
	Nodes      []*Node
	Expiration time.Time
}

func (p *Neighbors) sender() NodeID {
	return p.ID
}

func (n *Neighbors) serialize() ([]byte, error) {
	nodepbs := make([]*protos.Node, len(n.Nodes))
	for i, node := range n.Nodes {
		nodepbs[i] = node.proto()
	}

	neighborspb := &protos.Neighbors{
		ID:         n.ID[:],
		Nodes:      nodepbs,
		Expiration: n.Expiration.Unix(),
	}
	return proto.Marshal(neighborspb)
}

func (n *Neighbors) deserialize(buf []byte) (err error) {
	pb := new(protos.Neighbors)

	err = proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	id, err := Bytes2NodeID(pb.ID)
	if err != nil {
		return
	}
	n.ID = id

	n.Expiration = time.Unix(pb.Expiration, 0)

	nodes := make([]*Node, 0, len(pb.Nodes))

	for _, npb := range pb.Nodes {
		node, err := protoToNode(npb)
		if err != nil {
			nodes = append(nodes, node)
		}
	}

	n.Nodes = nodes

	return nil
}

func (p *Neighbors) pack(priv ed25519.PrivateKey) (data []byte, hash types.Hash, err error) {
	buf, err := p.serialize()
	if err != nil {
		return nil, hash, err
	}

	data, hash = composePacket(priv, neighborsCode, buf)
	return data, hash, nil
}

func (p *Neighbors) isExpired() bool {
	return isExpired(p.Expiration)
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

func unPacket(data []byte) (p *packet, err error) {
	pktVersion := data[0]

	if pktVersion != version {
		return nil, fmt.Errorf("unmatched version: %d / %d\n", pktVersion, version)
	}

	pktCode := packetCode(data[1])
	pktHash := data[2:34]
	payloadWithSig := data[34:]
	pktSig := data[34:98]
	payload := data[98:]

	// compare checksum
	reHash := crypto.Hash(32, payloadWithSig)
	if !bytes.Equal(reHash, pktHash) {
		return nil, errWrongHash
	}

	// unpack packet to get content and signature
	m, err := decode(pktCode, payload)
	if err != nil {
		return
	}

	// verify signature
	id := m.sender()
	valid, err := crypto.VerifySig(id[:], payload, pktSig)
	if err != nil {
		return
	}

	var hash types.Hash
	if valid {
		copy(hash[:], pktHash)

		return &packet{
			fromID: id,
			code:   pktCode,
			hash:   hash,
			msg:    m,
		}, nil
	}

	return nil, errInvalidSig
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
