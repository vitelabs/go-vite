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

const version byte = 2

var expiration = 20 * time.Second

func getExpiration() time.Time {
	return time.Now().Add(expiration)
}

func isExpired(t time.Time) bool {
	return t.Before(time.Now())
}

// packetCode
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
// consider varint encoding of protobuf, 1 byte origin data maybe take up 2 bytes space after encode.
// so the payload should small than 1200 bytes.
const maxPacketLength = 1200 // should small than MTU
const maxNeighborsNodes = 10

var errUnmatchedHash = errors.New("validate discovery packet error: invalid hash")
var errInvalidSig = errors.New("validate discovery packet error: invalid signature")

type packet struct {
	fromID NodeID
	from   *net.UDPAddr
	code   packetCode
	hash   types.Hash
	msg    Message
}

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
	pb := &protos.Ping{
		ID:         p.ID[:],
		IP:         p.IP,
		UDP:        uint32(p.UDP),
		TCP:        uint32(p.TCP),
		Expiration: p.Expiration.Unix(),
	}
	return proto.Marshal(pb)
}

func (p *Ping) deserialize(buf []byte) error {
	pb := new(protos.Ping)
	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	id, err := Bytes2NodeID(pb.ID)
	if err != nil {
		return err
	}

	p.ID = id
	p.IP = pb.IP
	p.UDP = uint16(pb.UDP)
	p.TCP = uint16(pb.TCP)
	p.Expiration = time.Unix(pb.Expiration, 0)

	return nil
}

func (p *Ping) pack(key ed25519.PrivateKey) (pkt []byte, hash types.Hash, err error) {
	payload, err := p.serialize()
	if err != nil {
		return
	}

	pkt, hash = composePacket(key, pingCode, payload)
	return
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
	pb := &protos.Pong{
		ID:         p.ID[:],
		Ping:       p.Ping[:],
		Expiration: p.Expiration.Unix(),
	}
	return proto.Marshal(pb)
}

func (p *Pong) deserialize(buf []byte) error {
	pb := new(protos.Pong)
	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	id, err := Bytes2NodeID(pb.ID)
	if err != nil {
		return err
	}

	p.ID = id
	copy(p.Ping[:], pb.Ping)
	p.Expiration = time.Unix(pb.Expiration, 0)

	return nil
}

func (p *Pong) pack(key ed25519.PrivateKey) (pkt []byte, hash types.Hash, err error) {
	payload, err := p.serialize()
	if err != nil {
		return
	}

	pkt, hash = composePacket(key, pongCode, payload)

	return
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

func (f *FindNode) sender() NodeID {
	return f.ID
}

func (f *FindNode) serialize() ([]byte, error) {
	pb := &protos.FindNode{
		ID:         f.ID[:],
		Target:     f.Target[:],
		Expiration: f.Expiration.Unix(),
	}
	return proto.Marshal(pb)
}

func (f *FindNode) deserialize(buf []byte) error {
	pb := &protos.FindNode{}
	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	id, err := Bytes2NodeID(pb.ID)
	if err != nil {
		return err
	}

	target, err := Bytes2NodeID(pb.Target)
	if err != nil {
		return err
	}

	f.ID = id
	f.Target = target
	f.Expiration = time.Unix(pb.Expiration, 0)

	return nil
}

func (f *FindNode) pack(priv ed25519.PrivateKey) (pkt []byte, hash types.Hash, err error) {
	payload, err := f.serialize()
	if err != nil {
		return
	}

	pkt, hash = composePacket(priv, findnodeCode, payload)

	return
}

func (f *FindNode) isExpired() bool {
	return isExpired(f.Expiration)
}

// @message neighbors
type Neighbors struct {
	ID         NodeID
	Nodes      []*Node
	Expiration time.Time
}

func (n *Neighbors) sender() NodeID {
	return n.ID
}

func (n *Neighbors) serialize() ([]byte, error) {
	npbs := make([]*protos.Node, len(n.Nodes))
	for i, node := range n.Nodes {
		npbs[i] = node.proto()
	}

	pb := &protos.Neighbors{
		ID:         n.ID[:],
		Nodes:      npbs,
		Expiration: n.Expiration.Unix(),
	}
	return proto.Marshal(pb)
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

	nodes := make([]*Node, 0, len(pb.Nodes))

	for _, npb := range pb.Nodes {
		node, err := protoToNode(npb)
		if err != nil {
			nodes = append(nodes, node)
		}
	}

	n.ID = id
	n.Expiration = time.Unix(pb.Expiration, 0)
	n.Nodes = nodes

	return nil
}

func (n *Neighbors) pack(priv ed25519.PrivateKey) (pkt []byte, hash types.Hash, err error) {
	payload, err := n.serialize()
	if err != nil {
		return
	}

	pkt, hash = composePacket(priv, neighborsCode, payload)
	return
}

func (n *Neighbors) isExpired() bool {
	return isExpired(n.Expiration)
}

// version code checksum signature payload
func composePacket(priv ed25519.PrivateKey, code packetCode, payload []byte) (data []byte, hash types.Hash) {
	sig := ed25519.Sign(priv, payload)
	checksum := crypto.Hash(32, append(sig, payload...))

	chunk := [][]byte{
		{version, byte(code)},
		checksum,
		sig,
		payload,
	}

	data = bytes.Join(chunk, []byte{})

	copy(hash[:], checksum)
	return data, hash
}

func unPacket(data []byte) (p *packet, err error) {
	pktVersion := data[0]

	if pktVersion != version {
		return nil, fmt.Errorf("unmatched discovery packet version: received packet version is %d, but we are %d \n", pktVersion, version)
	}

	pktCode := packetCode(data[1])
	pktHash := data[2:34] // 32 bytes
	payloadWithSig := data[34:]
	pktSig := data[34:98] // 64 bytes
	payload := data[98:]

	// compare checksum
	reHash := crypto.Hash(32, payloadWithSig)
	if !bytes.Equal(reHash, pktHash) {
		return nil, errUnmatchedHash
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

	if valid {
		hash := new(types.Hash)

		err = hash.SetBytes(pktHash)
		if err != nil {
			return
		}

		p = &packet{
			fromID: id,
			from:   nil,
			code:   pktCode,
			hash:   *hash,
			msg:    m,
		}

		return p, nil
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
