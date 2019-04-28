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
	"sync"
	"time"

	"github.com/vitelabs/go-vite/p2p/vnode"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/p2p/discovery/protos"
)

const version byte = 0
const expiration = 5 * time.Second
const maxPacketLength = 1200
const signatureLength = 64
const packetHeadLength = 1 + 1 + len(vnode.ZERO)
const minPacketLength = packetHeadLength + signatureLength // 98
const maxPayloadLength = maxPacketLength - minPacketLength // 1102

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
	payload, err := m.body.serialize()
	if err != nil {
		return
	}

	data, hash = pack(key, m.c, payload)
	return
}

type ping struct {
	from, to *vnode.EndPoint
	net      int
	ext      []byte
	time     time.Time
}

func (p *ping) serialize() (data []byte, err error) {
	pb := &protos.Ping{
		Net:  uint32(p.net),
		Ext:  p.ext,
		Time: p.time.Unix(),
	}

	var buf []byte
	if p.from == nil {
		pb.From = nil
	} else {
		buf, err = p.from.Serialize()
		if err != nil {
			pb.From = nil
		} else {
			pb.From = buf
		}
	}

	if p.to == nil {
		pb.To = nil
	} else {
		buf, err = p.to.Serialize()
		if err != nil {
			pb.To = nil
		} else {
			pb.To = buf
		}
	}

	return proto.Marshal(pb)
}

func (p *ping) deserialize(buf []byte) error {
	pb := new(protos.Ping)
	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	p.net = int(pb.Net)
	p.ext = pb.Ext
	p.time = time.Unix(pb.Time, 0)

	from := new(vnode.EndPoint)
	err = from.Deserialize(pb.From)
	if err != nil {
		p.from = nil
	} else {
		p.from = from
	}

	to := new(vnode.EndPoint)
	err = to.Deserialize(pb.To)
	if err != nil {
		p.to = nil
	} else {
		p.to = to
	}

	return nil
}

func (p *ping) expired() bool {
	return time.Now().Sub(p.time) > expiration
}

type pong struct {
	from, to *vnode.EndPoint
	net      int
	ext      []byte
	echo     []byte
	time     time.Time
}

func (p *pong) serialize() (data []byte, err error) {
	pb := &protos.Pong{}

	pb.Net = uint32(p.net)
	pb.Ext = p.ext
	pb.Echo = p.echo
	pb.Time = p.time.Unix()

	var buf []byte
	if p.from == nil {
		pb.From = nil
	} else {
		buf, err = p.from.Serialize()
		if err != nil {
			pb.From = nil
		} else {
			pb.From = buf
		}
	}

	if p.to == nil {
		pb.To = nil
	} else {
		buf, err = p.to.Serialize()
		if err != nil {
			pb.To = nil
		} else {
			pb.To = buf
		}
	}

	return proto.Marshal(pb)
}

func (p *pong) deserialize(data []byte) error {
	pb := new(protos.Pong)
	err := proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}

	p.net = int(pb.Net)
	p.ext = pb.Ext
	p.echo = pb.Echo
	p.time = time.Unix(pb.Time, 0)

	from := new(vnode.EndPoint)
	err = from.Deserialize(pb.From)
	if err != nil {
		p.from = nil
	} else {
		p.from = from
	}

	to := new(vnode.EndPoint)
	err = to.Deserialize(pb.To)
	if err != nil {
		p.to = nil
	} else {
		p.to = to
	}

	return nil
}

func (p *pong) expired() bool {
	return time.Now().Sub(p.time) > expiration
}

type findnode struct {
	target vnode.NodeID
	count  int
	time   time.Time
}

func (f *findnode) serialize() ([]byte, error) {
	pb := &protos.Findnode{
		Target: f.target.Bytes(),
		Count:  uint32(f.count),
		Time:   f.time.Unix(),
	}
	return proto.Marshal(pb)
}

func (f *findnode) deserialize(buf []byte) error {
	pb := &protos.Findnode{}
	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	f.target, err = vnode.Bytes2NodeID(pb.Target)
	if err != nil {
		return err
	}

	f.count = int(pb.Count)
	f.time = time.Unix(pb.Time, 0)

	return nil
}

func (f *findnode) expired() bool {
	return time.Now().Sub(f.time) > expiration
}

type neighbors struct {
	endpoints []*vnode.EndPoint
	last      bool
	time      time.Time
}

func (n *neighbors) serialize() ([]byte, error) {
	pb := &protos.Neighbors{
		Last: n.last,
		Time: n.time.Unix(),
	}

	pb.Nodes = make([][]byte, 0, len(n.endpoints))
	for _, ep := range n.endpoints {
		buf, err := ep.Serialize()
		if err != nil {
			continue
		}
		pb.Nodes = append(pb.Nodes, buf)
	}

	return proto.Marshal(pb)
}

func (n *neighbors) deserialize(buf []byte) (err error) {
	pb := new(protos.Neighbors)

	err = proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	n.last = pb.Last
	n.time = time.Unix(pb.Time, 0)

	n.endpoints = make([]*vnode.EndPoint, 0, len(pb.Nodes))
	for _, buf = range pb.Nodes {
		e := new(vnode.EndPoint)
		err = e.Deserialize(buf)
		if err != nil {
			continue
		}
		n.endpoints = append(n.endpoints, e)
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

var packet_pool = sync.Pool{
	New: func() interface{} {
		return new(packet)
	},
}

func retrievePacket() *packet {
	v := packet_pool.Get().(*packet)
	return v
}

func recyclePacket(pkt *packet) {
	packet_pool.Put(pkt)
}

// unPacket []byte to message
func unPacket(data []byte, p *packet) (err error) {
	if len(data) < minPacketLength {
		err = errPacketTooSmall
		return
	}

	if data[0] != version {
		err = errDiffVersion
		return
	}

	c := data[1]
	id := data[2:packetHeadLength]
	payload := data[packetHeadLength : len(data)-signatureLength]
	signature := data[len(data)-signatureLength:]

	valid, err := crypto.VerifySig(id, payload, signature)
	if err != nil {
		return
	}
	if !valid {
		err = errInvalidSignature
		return
	}

	fromId, err := vnode.Bytes2NodeID(id)
	if err != nil {
		return
	}

	// unpack packet to get content and signature
	m, err := decode(c, payload)
	if err != nil {
		return
	}

	p.c = c
	p.id = fromId
	p.body = m
	p.hash = crypto.Hash256(payload)

	return nil
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
