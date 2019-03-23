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

const IDBits = 32 * 8

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

// RandFromDistance will generate a random NodeID which satisfy `Distance(id, rid) == d`
func RandFromDistance(id NodeID, d uint) (rid NodeID) {
	if d == 0 {
		return id
	}

	if d >= IDBits {
		d = IDBits
	}

	rand.Read(rid[:])

	n := IDBits - d
	byt := n / 8
	bit := n % 8

	for i := uint(0); i < byt; i++ {
		rid[i] = id[i]
	}

	// set left bits to the same with id
	// make rid[i] left bits to zero
	// make id[i] right (8 - bits) to zero
	rid[byt] = (rid[byt] << bit >> bit) | (id[byt] >> (8 - bit) << (8 - bit))

	// must set the bits+1 different
	nextBitFromRight := 7 - bit
	theSrcBit := id[byt] & (byte(1) << nextBitFromRight)

	if theSrcBit == 1 {
		// set the bit of rid to 0
		rid[byt] = rid[byt] & (^(byte(1) << nextBitFromRight))
	} else {
		// set the bit of rid to 1
		rid[byt] = rid[byt] | (byte(1) << nextBitFromRight)
	}

	return
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

// Distance is bit-count minus the common bits between a and b, from left to right continuously, eg:
// a: 0000 1111
// b: 0100 0011
// Distance(a, b) == 8 - 1
func Distance(a, b NodeID) uint {
	if a == b {
		return IDBits
	}

	var n uint

	// common bytes
	for n = 0; n < uint(len(a)); n++ {
		if a[n] != b[n] {
			break
		}
	}

	// common bits
	xor := a[n] ^ b[n]

	n *= 8

	for (xor & 128) == 0 {
		n++
		xor <<= 1
	}

	return IDBits - n
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
	if !n.ID.IsZero() {
		str += n.ID.String()
	}

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

// ParseNode parse a string to Node
// Return error if missing Hostname/IP
func ParseNode(u string) (n *Node, err error) {
	index := strings.IndexRune(u, '@')
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
