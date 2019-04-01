/*
 * Copyright 2019 The go-vite Authors
 * This file is part of the go-vite library.
 *
 * The go-vite library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The go-vite library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with the go-vite library. If not, see <http://www.gnu.org/licenses/>.
 */

package discovery

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/vitelabs/go-vite/p2p/vnode"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/p2p/discovery/protos"
)

const version byte = 0
const expiration = 4 * time.Second
const maxPacketLength = 1200
const signatureLength = 64
const packetHeadLength = 1 + 1 + len(vnode.ZERO)
const minPacketLength = packetHeadLength + signatureLength
const maxPayloadLength = maxPacketLength - minPacketLength

type code = byte

const (
	codePing code = iota
	codePong
	codeFindnode
	codeNeighbors
	codeException
)

var errDiffVersion = errors.New("different packet version")
var errPacketTooSmall = errors.New("packet is too small")
var errInvalidSignature = errors.New("validate discovery packet error: invalid signature")

type body interface {
	serialize() ([]byte, error)
	deserialize([]byte) error
	expired() bool
}

type message struct {
	c  code
	id vnode.NodeID
	body
}

func (m message) pack(key ed25519.PrivateKey) (data, hash []byte, err error) {
	data, err = m.body.serialize()
	if err != nil {
		return
	}

	data, hash = pack(key, m.c, data)
	return
}

type ping struct {
	from, to *vnode.EndPoint
	net      uint32
	ext      []byte
	time     time.Time
}

func (p *ping) serialize() ([]byte, error) {
	from, _ := p.from.Serialize()
	to, _ := p.to.Serialize()

	pb := &protos.Ping{
		From: from,
		To:   to,
		Net:  p.net,
		Ext:  p.ext,
		Time: p.time.Unix(),
	}
	return proto.Marshal(pb)
}

func genNodeFromPing(res *packet) *Node {
	p := res.body.(*ping)

	var addr *net.UDPAddr
	var e vnode.EndPoint
	if p.from != nil {
		var err error
		addr, err = net.ResolveUDPAddr("udp", p.from.String())
		if err != nil {
			addr = res.from
			// generate from remote address
			e = vnode.FromUDPAddr(res.from)
		} else {
			e = *p.from
		}
	} else {
		addr = res.from
		e = vnode.FromUDPAddr(res.from)
	}

	return &Node{
		Node: vnode.Node{
			ID:       res.id,
			EndPoint: e,
			Net:      p.net,
			Ext:      p.ext,
		},
		addAt:    time.Now(),
		activeAt: time.Now(),
		addr:     addr,
	}
}

func updateFromPong(old *Node, res *packet) {

}

func (p *ping) deserialize(buf []byte) error {
	pb := new(protos.Ping)
	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	return nil
}

func (p *ping) expired() bool {
	return time.Now().Sub(p.time) > expiration
}

type pong struct {
	from, to *vnode.EndPoint
	net      uint32
	ext      []byte
	echo     []byte
	time     time.Time
}

func (p *pong) serialize() ([]byte, error) {
	pb := &protos.Pong{}
	return proto.Marshal(pb)
}

func (p *pong) deserialize(buf []byte) error {
	pb := new(protos.Pong)
	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	// todo

	return nil
}

func (p *pong) expired() bool {
	return time.Now().Sub(p.time) > expiration
}

type findnode struct {
	target vnode.NodeID
	count  uint32
	time   time.Time
}

func (f *findnode) serialize() ([]byte, error) {
	pb := &protos.Findnode{}
	return proto.Marshal(pb)
}

func (f *findnode) deserialize(buf []byte) error {
	pb := &protos.Findnode{}
	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	return nil
}

func (f *findnode) expired() bool {
	return time.Now().Sub(f.time) > expiration
}

type neighbors struct {
	endpoints []vnode.EndPoint
	last      bool
	time      time.Time
}

func (n *neighbors) serialize() ([]byte, error) {
	pb := &protos.Neighbors{}
	return proto.Marshal(pb)
}

func (n *neighbors) deserialize(buf []byte) (err error) {
	pb := new(protos.Neighbors)

	err = proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	return nil
}

func (n *neighbors) expired() bool {
	return time.Now().Sub(n.time) > expiration
}

// pack a message to []byte, structure as following:
// +---------+---------+---------------+--------------------------------------+---------------------------+
// | version |   code  |    nodeID     |              payload                 |         signature         |
// |  1 byte |  1 byte |    32 bytes   |            0 ~ 1102 bytes            |          64 bytes         |
// +---------+---------+---------------+--------------------------------------+---------------------------+
// return the packet []byte, and hash of the payload
func pack(priv ed25519.PrivateKey, code code, payload []byte) (data, hash []byte) {
	hash = crypto.Hash256(payload)

	signature := ed25519.Sign(priv, payload)

	chunk := [][]byte{
		{version, code},
		priv.PubByte(),
		payload,
		signature,
	}

	data = bytes.Join(chunk, nil)

	return
}

// unPacket []byte to message
func unPacket(data []byte) (p *packet, err error) {
	if len(data) < minPacketLength {
		return nil, errPacketTooSmall
	}

	if data[0] != version {
		return nil, errDiffVersion
	}

	c := data[1]
	id := data[2:packetHeadLength]
	payload := data[packetHeadLength : len(data)-signatureLength]
	signature := data[len(data)-signatureLength:]

	valid, err := crypto.VerifySig(id, payload, signature)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, errInvalidSignature
	}

	fromId, err := vnode.Bytes2NodeID(id)
	if err != nil {
		return nil, err
	}

	// unpack packet to get content and signature
	m, err := decode(c, payload)
	if err != nil {
		return
	}

	p = &packet{
		message: message{
			c:    c,
			id:   fromId,
			body: m,
		},
	}

	return p, nil
}

func decode(code code, payload []byte) (m body, err error) {
	switch code {
	case codePing:
		m = new(ping)
	case codePong:
		m = new(pong)
	case codeFindnode:
		m = new(findnode)
	case codeNeighbors:
		m = new(neighbors)
	default:
		return m, fmt.Errorf("decode packet error: unknown code %d", code)
	}

	err = m.deserialize(payload)

	return m, err
}
