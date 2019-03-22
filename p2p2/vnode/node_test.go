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
	"bytes"
	"fmt"
	"math/rand"
	"testing"
)

func compare(n, n2 *Node, rest bool) error {
	if n.ID != n2.ID {
		return fmt.Errorf("different ID %s %s", n.ID, n2.ID)
	}

	if !bytes.Equal(n.Host, n2.Host) {
		return fmt.Errorf("different Host %v %v", n.Host, n2.Host)
	}

	if n.Port != n2.Port {
		return fmt.Errorf("different Port %d %d", n.Port, n2.Port)
	}

	if n.Net != n2.Net {
		return fmt.Errorf("different Net %d %d", n.Net, n2.Net)
	}

	if rest && !bytes.Equal(n.Ext, n2.Ext) {
		return fmt.Errorf("different Ext %v %v", n.Ext, n2.Ext)
	}

	return nil
}

func TestNodeID_IsZero(t *testing.T) {
	if !ZERO.IsZero() {
		t.Fail()
	}
}

func TestNode_Serialize(t *testing.T) {
	var conds = [4][2]bool{
		{true, true},
		{true, false},
		{false, false},
		{false, true},
	}

	for i := 0; i < 4; i++ {
		n := MockNode(conds[i][0], conds[i][1])
		data, err := n.Serialize()
		if err != nil {
			t.Error(err)
		}

		n2 := new(Node)
		err = n2.Deserialize(data)
		if err != nil {
			t.Error(err)
		}

		if err = compare(n, n2, true); err != nil {
			t.Error(err)
		}
	}
}

func TestParseNode_Protocol(t *testing.T) {
	var protocolTests = [...]string{
		"vnode://vite.org",
		"https://vite.org",
		"http://vite.org",
		"vite.org",
	}

	var err, err2 error
	for _, url := range protocolTests {
		_, err2 = ParseNode(url)
		if err != err2 {
			t.Error(err2)
		}
		err = errInvalidScheme
	}
}

func TestParseNode_Host(t *testing.T) {
	factor := func(host []byte, typ HostType, port int) func(n *Node) error {
		return func(n *Node) error {
			if !bytes.Equal(n.Host, host) {
				return fmt.Errorf("different host: %v %v", n.Host, host)
			}
			if n.Port != port {
				return fmt.Errorf("different port: %d %d", n.Port, port)
			}
			if n.typ != typ {
				return fmt.Errorf("different type: %d %d", n.typ, typ)
			}
			return nil
		}
	}

	var protocolTests = [...]struct {
		url    string
		handle func(n *Node) error
	}{
		{
			"vnode://vite.org",
			factor([]byte("vite.org"), HostDomain, DefaultPort),
		},
		{
			"vnode://vite.org:8888",
			factor([]byte("vite.org"), HostDomain, 8888),
		},

		{
			"vnode://127.0.0.1",
			factor([]byte{127, 0, 0, 1}, HostIPv4, DefaultPort),
		},
		{
			"vnode://127.0.0.1:8888",
			factor([]byte{127, 0, 0, 1}, HostIPv4, 8888),
		},
	}

	var node *Node
	var err error
	for _, tt := range protocolTests {
		node, err = ParseNode(tt.url)
		if err != nil {
			t.Error(err)
		} else {
			if err = tt.handle(node); err != nil {
				t.Error(tt.url, err)
			} else {
				if tt.url != node.String() {
					t.Errorf("different url: %s %s", tt.url, node.String())
				}
			}
		}
	}
}

func TestParseNode_Net(t *testing.T) {
	n, err := ParseNode("vnode://vite.org/2")

	if err != nil {
		t.Error(err)
	}

	if n.Net != 2 {
		t.Fail()
	}

	n, err = ParseNode("vnode://vite.org")
	if err != nil {
		t.Error(err)
	}
	if n.Net != 0 {
		t.Fail()
	}
}

func TestCommonBits(t *testing.T) {
	var a, b NodeID
	total := len(a)

	// change last byte
	b[total-1] = b[total-1] | 1

	if Distance(a, b) != 8*total-1 {
		t.Errorf("distance should be %d, but %d", 8*total-1, Distance(a, b))
	}

	// reset
	b = NodeID{}

	// change first byte
	b[0] = b[0] | 1
	if Distance(a, b) != 7 {
		t.Errorf("distance should be %d, but %d", 7, Distance(a, b))
	}

	// reset
	b = NodeID{}

	// change random byte
	bytIndex := rand.Intn(total)
	bitIndex := rand.Intn(7) + 1
	b[bytIndex] = 1 << uint(bitIndex)
	if Distance(a, b) != 8*bytIndex+(8-bitIndex-1) {
		t.Errorf("distance should be %d, but %d", 8*bytIndex+(8-bitIndex-1), Distance(a, b))
	}
}
