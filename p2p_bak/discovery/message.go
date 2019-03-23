package discovery

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/vitelabs/go-vite/p2p/network"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/p2p/discovery/protos"
)

const version byte = 2

var expiration = 10 * time.Second

func getExpiration() time.Time {
	return time.Now().Add(2 * expiration)
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
	exceptionCode
)

var packetStrs = [...]string{
	"ping",
	"pong",
	"findnode",
	"neighbors",
	"exception",
}

func (c packetCode) String() string {
	if c > exceptionCode {
		return "unknown packet code"
	}
	return packetStrs[c]
}

// the full packet must be little than 1400bytes, consist of:
// version(1 byte), code(1 byte), checksum(32 bytes), signature(64 bytes), payload
// consider varint encoding of protobuf, 1 byte origin data maybe take up 2 bytes space after encode.
// so the payload should small than 1200 bytes.
const maxPacketLength = 1200 // should small than MTU

var errUnmatchedHash = errors.New("validate discovery packet error: invalid hash")
var errInvalidSig = errors.New("validate discovery packet error: invalid signature")

type Message interface {
	serialize() ([]byte, error)
	deserialize([]byte) error
	pack(ed25519.PrivateKey) ([]byte, types.Hash, error)
	isExpired() bool
	sender() NodeID
	String() string
}

type Ping struct {
	ID         NodeID
	TCP        uint16
	Expiration time.Time
	Net        network.ID
	Ext        []byte
}

func (p *Ping) sender() NodeID {
	return p.ID
}

func (p *Ping) serialize() ([]byte, error) {
	pb := &protos.Ping{
		ID:         p.ID[:],
		TCP:        uint32(p.TCP),
		Expiration: p.Expiration.Unix(),
		Net:        uint32(p.Net),
		Ext:        p.Ext,
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
	p.TCP = uint16(pb.TCP)
	p.Expiration = time.Unix(pb.Expiration, 0)
	p.Net = network.ID(pb.Net)
	p.Ext = pb.Ext

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

func (p *Ping) String() string {
	return "ping"
}

type Pong struct {
	ID         NodeID
	Ping       types.Hash
	IP         net.IP
	Expiration time.Time
}

func (p *Pong) sender() NodeID {
	return p.ID
}

func (p *Pong) serialize() ([]byte, error) {
	pb := &protos.Pong{
		ID:         p.ID[:],
		Ping:       p.Ping[:],
		IP:         p.IP,
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
	p.IP = pb.IP
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

func (p *Pong) String() string {
	return "pong<" + p.Ping.String() + ">"
}

type FindNode struct {
	ID         NodeID
	Target     NodeID
	Expiration time.Time
	N          uint32
}

func (f *FindNode) sender() NodeID {
	return f.ID
}

func (f *FindNode) serialize() ([]byte, error) {
	pb := &protos.FindNode{
		ID:         f.ID[:],
		Target:     f.Target[:],
		Expiration: f.Expiration.Unix(),
		N:          f.N,
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
	f.N = pb.N

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

func (f *FindNode) String() string {
	return "findnode<" + f.Target.String() + ">"
}

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
		if err == nil {
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

func (n *Neighbors) String() string {
	return "neighbors<" + strconv.Itoa(len(n.Nodes)) + ">"
}

// @section Exception
type eCode uint64

const (
	eCannotUnpack eCode = iota + 1
	eLowVersion
	eBlocked
	eUnKnown
)

var eMsg = [...]string{
	eCannotUnpack: "can`t unpack message, maybe not the same discovery version",
	eLowVersion:   "your version is lower, maybe you should upgrade",
	eBlocked:      "you are blocked",
	eUnKnown:      "unknown discovery message",
}

func (e eCode) String() string {
	return eMsg[e]
}

type Exception struct {
	Code eCode
}

func (e *Exception) serialize() ([]byte, error) {
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, uint64(e.Code))
	return buf[:n], nil
}

func (e *Exception) deserialize(buf []byte) error {
	code, n := binary.Varint(buf)
	if n <= 0 {
		return errors.New("parse error")
	}
	e.Code = eCode(code)
	return nil
}

func (e *Exception) pack(priv ed25519.PrivateKey) (pkt []byte, hash types.Hash, err error) {
	payload, err := e.serialize()
	if err != nil {
		return
	}

	pkt, hash = composePacket(priv, exceptionCode, payload)
	return
}

func (e *Exception) isExpired() bool {
	return false
}

func (e *Exception) sender() (id NodeID) {
	return
}

func (n *Exception) String() string {
	return "exception<" + n.Code.String() + ">"
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

var errPktSmall = errors.New("packet is too small")

func unPacket(data []byte) (p *packet, err error) {
	pktVersion := data[0]

	if pktVersion != version {
		return nil, fmt.Errorf("unmatched discovery packet version: received packet version is %d, but we are %d", pktVersion, version)
	}

	if len(data) < 99 {
		return nil, errPktSmall
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
