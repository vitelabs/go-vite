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

package vnode

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/p2p/vnode/protos"
)

const DefaultPort = 8483
const PortLength = 2
const MaxHostLength = 1<<6 - 1

var errUnmatchedLength = errors.New("needs 64 hex chars")

const idBytes = 32
const IDBits = idBytes * 8

// ZERO is the zero-value of NodeID type
var ZERO NodeID

// NodeID use to mark node, and build a structural network
type NodeID [IDBits / 8]byte

// String return a hex coded string of NodeID
func (id NodeID) String() string {
	return hex.EncodeToString(id[:])
}

func (id NodeID) Brief() string {
	return id.String()[:8]
}

func (id NodeID) Bytes() []byte {
	return id[:]
}

// IsZero validate whether a NodeID is zero-value
func (id NodeID) IsZero() bool {
	for _, byt := range id {
		if byt|0 != 0 {
			return false
		}
	}
	return true
}

func RandomNodeID() (id NodeID) {
	_, _ = rand.Read(id[:])
	return
}

// RandFromDistance will generate a random NodeID which satisfy `Distance(id, rid) == d`
func RandFromDistance(id NodeID, d uint) (rid NodeID) {
	if d >= IDBits {
		d = IDBits
	}

	return commonNBits(id, IDBits-d)
}

// Hex2NodeID parse a hex coded string to NodeID
func Hex2NodeID(str string) (id NodeID, err error) {
	bytes, err := hex.DecodeString(strings.TrimPrefix(str, "0x"))
	if err != nil {
		return
	}

	return Bytes2NodeID(bytes)
}

// Bytes2NodeID turn a slice to NodeID
func Bytes2NodeID(buf []byte) (id NodeID, err error) {
	if len(buf) != len(id) {
		return id, errUnmatchedLength
	}

	copy(id[:], buf)
	return
}

func commonBits(a, b NodeID) (n uint) {
	for n = 0; n < idBytes; n++ {
		if a[n] != b[n] {
			break
		}
	}

	if n == idBytes {
		return IDBits
	}

	// common bits
	xor := a[n] ^ b[n]

	n *= 8

	for (xor & 128) == 0 {
		n++
		xor <<= 1
	}

	return
}

func commonNBits(a NodeID, n uint) (b NodeID) {
	if n == IDBits {
		return a
	}

	b = RandomNodeID()

	byt := n / 8
	bit := n % 8

	for i := uint(0); i < byt; i++ {
		b[i] = a[i]
	}

	// reset left bit bits
	b[byt] = b[byt] << bit >> bit
	// set left bit bits same with a
	b[byt] |= a[byt] >> (8 - bit) << (8 - bit)

	diffBita := a[byt] << bit >> 7 << (7 - bit)
	if diffBita > 0 {
		// set diffBitb to 0
		b[byt] &= 255 - (1 << (7 - bit))
	} else {
		// set diffBitb to 1
		b[byt] |= 1 << (7 - bit)
	}

	return
}

// Distance is bit-count minus the common bits, from left to right continuously, between a and b. eg:
// a: 0000 1111
// b: 0100 0011
// common bits between a and b is 1
// Distance(a, b) == 8 - 1
func Distance(a, b NodeID) uint {
	return IDBits - commonBits(a, b)
}

var errMissHost = errors.New("missing Host")

// Node mean a node in vite P2P network
type Node struct {
	// ID is the unique node identity
	ID NodeID

	EndPoint

	// Net is the network this node belongs
	Net uint32
	// Ext can be arbitrary data, will be sent to other nodes
	Ext []byte
}

// Address is formatted `domain:Port` or `IP:Port`
func (n Node) Address() string {
	return n.EndPoint.String()
}

// String marshal node to string, looks like:
// <hex_node_id>@domain:<port>/net
// <hex_node_id>@[IP]:<port>/net
// the field `Ext` will not be included in the encoded string
func (n Node) String() (str string) {
	str = n.ID.String() + "@"

	if n.Port == DefaultPort {
		str += n.Hostname()
	} else {
		str += n.Address()
	}

	if n.Net > 0 {
		str += "/" + strconv.Itoa(int(n.Net))
	}

	return
}

const protocol = "vnode://"

// ParseNode parse a string to Node
// Return error if missing Hostname/IP
func ParseNode(u string) (n *Node, err error) {
	var index int
	if index = strings.Index(u, protocol); index > -1 {
		index += 8
		u = u[index:]
	}

	index = strings.IndexRune(u, '@')
	if index == len(u)-1 {
		err = errMissHost
		return
	}

	n = new(Node)

	if index > 0 {
		n.ID, err = Hex2NodeID(u[:index])
		if err != nil {
			return
		}
	}

	u = u[index+1:]

	host := u
	index = strings.IndexRune(u, '/')
	if index > 0 {
		host = u[:index]
	}

	n.EndPoint, err = ParseEndPoint(host)
	if err != nil {
		return
	}

	if index > 0 && index < len(u)-1 {
		n.Net, err = parseNid(u[index+1:])
		if err != nil {
			return
		}
	}

	return
}

func parsePort(str string) (port int, err error) {
	p, err := strconv.ParseUint(str, 10, 16)
	if err != nil {
		return
	}

	return int(p), nil
}

func parseNid(str string) (nid uint32, err error) {
	n, err := strconv.ParseInt(str, 10, 8)
	if err != nil {
		return
	}

	return uint32(n), nil
}

func (n *Node) Serialize() ([]byte, error) {
	pb := &protos.Node{
		ID:       n.ID.Bytes(),
		Hostname: n.Host,
		HostType: uint32(n.Typ),
		Port:     uint32(n.Port),
		Net:      n.Net,
		Ext:      n.Ext,
	}

	return proto.Marshal(pb)
}

func (n *Node) Deserialize(data []byte) (err error) {
	pb := new(protos.Node)
	err = proto.Unmarshal(data, pb)
	if err != nil {
		return
	}

	n.ID, err = Bytes2NodeID(pb.ID)
	if err != nil {
		return
	}

	n.Host = pb.Hostname
	n.Typ = HostType(pb.HostType)

	n.Port = int(pb.Port)

	n.Net = pb.Net

	n.Ext = pb.Ext

	return
}
