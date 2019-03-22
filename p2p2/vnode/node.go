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
	"encoding/hex"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/p2p2/vnode/protos"
)

const DefaultPort = 8483
const PortLength = 2
const MaxHostLength = 1<<6 - 1

var errUnmatchedLength = errors.New("needs 64 hex chars")

// ZERO is the zero-value of NodeID type
var ZERO NodeID

// NodeID use to mark node, and build a structural network
type NodeID [32]byte

// String return a hex coded string of NodeID
func (id NodeID) String() string {
	return hex.EncodeToString(id[:])
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

// Distance is the common bits between a and b, from left to right continuously, eg:
// 0000 1111
// 0100 0011
// Distance is 1
func Distance(a, b NodeID) (n int) {
	if a == b {
		return 8 * len(a)
	}

	// common bytes
	for n = range a {
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

	return
}

var errMissHost = errors.New("missing Host")
var errInvalidScheme = errors.New("invalid scheme")

// Node mean a node in vite P2P network
type Node struct {
	// ID is the unique node identity
	ID NodeID

	EndPoint

	// Net is the network this node belongs
	Net int
	// Ext can be arbitrary data, will be sent to other nodes
	Ext []byte
}

// Address is formatted `domain:Port` or `IP:Port`
func (n Node) Address() string {
	return n.EndPoint.String()
}

const NodeURLScheme = "vnode"

// String marshal node to url-like string, looks like:
// vnode://<hex_node_id>@<ip/domain>:<Port>/net
// the field `Ext` will not be included in the encoded string
func (n Node) String() string {
	var str = NodeURLScheme + "://"

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

	return str
}

// ParseNode parse a url-like string to Node
// Return error if missing Hostname/IP, or Scheme not equal "vnode"
func ParseNode(u string) (n *Node, err error) {
	nodeURL, err := url.Parse(u)
	if err != nil {
		return
	}

	if nodeURL.Scheme != NodeURLScheme {
		err = errInvalidScheme
		return
	}

	n = new(Node)

	if user := nodeURL.User; user != nil {
		n.ID, err = Hex2NodeID(user.Username())
		if err != nil {
			return
		}
	}

	n.Host, n.typ, err = parseHost(nodeURL.Hostname())
	if err != nil {
		return
	}

	port := nodeURL.Port()
	if port == "" {
		n.Port = DefaultPort
	} else if n.Port, err = parsePort(nodeURL.Port()); err != nil {
		return
	}

	// Path include "/"
	if len(nodeURL.Path) > 1 {
		n.Net, err = parseNid(nodeURL.Path[1:])
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

func parseNid(str string) (nid int, err error) {
	n, err := strconv.ParseInt(str, 10, 8)
	if err != nil {
		return
	}

	return int(n), nil
}

func (n *Node) Serialize() ([]byte, error) {
	pb := &protos.Node{
		ID:       n.ID.Bytes(),
		Hostname: n.Host,
		HostType: uint32(n.typ),
		Port:     uint32(n.Port),
		Net:      uint32(n.Net),
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
	n.typ = HostType(pb.HostType)

	n.Port = int(pb.Port)

	n.Net = int(pb.Net)

	n.Ext = pb.Ext

	return
}
