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
	crand "crypto/rand"
	"errors"
	"fmt"
	"testing"
)

func compare(n, n2 *Node, ext bool) error {
	if n.ID != n2.ID {
		return fmt.Errorf("different ID %s %s", n.ID, n2.ID)
	}

	if !bytes.Equal(n.Host, n2.Host) {
		return fmt.Errorf("different Host %v %v", n.Host, n2.Host)
	}

	if n.Port != n2.Port {
		return fmt.Errorf("different Port %d %d", n.Port, n2.Port)
	}

	if n.Typ != n2.Typ {
		return fmt.Errorf("different type %d %d", n.Typ, n2.Typ)
	}

	if n.Net != n2.Net {
		return fmt.Errorf("different Net %d %d", n.Net, n2.Net)
	}

	if ext && !bytes.Equal(n.Ext, n2.Ext) {
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

func TestParseNode_Net(t *testing.T) {
	n, err := ParseNode("vite.org/2")

	if err != nil {
		t.Error(err)
	}

	if n.Net != 2 {
		t.Fail()
	}

	n, err = ParseNode("vite.org")
	if err != nil {
		t.Error(err)
	}
	if n.Net != 0 {
		t.Fail()
	}
}

func TestCommonBits(t *testing.T) {
	var id, id2 NodeID

	if commonBits(id, id2) != IDBits {
		t.Error("not equal")
	}

	for i := uint(0); i < IDBits; i++ {
		var a, b NodeID

		byt := i / 8
		bit := i % 8

		// set i bit to 1
		a[byt] = a[byt] | (1 << (7 - bit))

		if commonBits(a, b) != i {
			t.Errorf("commonBits should equal %d, but get %d", i, commonBits(a, b))
		}
	}
}

func TestCommonNBits(t *testing.T) {
	var id = RandomNodeID()

	for i := uint(0); i <= IDBits; i++ {
		id2 := commonNBits(id, i)
		if commonBits(id, id2) != i {
			commonNBits(id, i)
			t.Errorf(fmt.Sprintf("commonBits should equal %d, but get %d", i, commonBits(id, id2)))
		}
	}
}

func TestDistance(t *testing.T) {
	var a, b NodeID
	if Distance(a, b) != 0 {
		t.Errorf("distance should equal %d, but get %d", 0, Distance(a, b))
	}

	for i := uint(0); i < IDBits; i++ {
		var a, b NodeID

		byt := i / 8
		bit := i % 8

		// set i bit to 1
		a[byt] = a[byt] | (1 << (7 - bit))

		if Distance(a, b) != IDBits-i {
			t.Errorf("distance should equal %d, but get %d", i, Distance(a, b))
		}
	}
}

func TestRandFromDistance(t *testing.T) {
	for i := 0; i < 1000; i++ {
		var id NodeID
		_, err := crand.Read(id[:])
		if err != nil {
			t.Error(err)
		}

		for d := uint(0); d <= idBytes; d++ {
			id2 := RandFromDistance(id, d)
			if Distance(id, id2) != d {
				t.Errorf("distance should equal %d, but get %d", d, Distance(id, id2))
			}
		}
	}
}

func TestParseNode(t *testing.T) {
	type sample struct {
		url   string
		equal func(n *Node, err error) error
	}

	var samples = []sample{
		{
			"vite.org",
			func(n *Node, err error) error {
				if err != nil {
					return errors.New("error should be nil")
				}
				if n.ID != ZERO {
					return fmt.Errorf("id should be zero, not %s", n.ID.String())
				}
				if n.EndPoint.String() != "vite.org:8483" {
					return fmt.Errorf("endpoint should not be %s", n.EndPoint.String())
				}
				if n.Typ != HostDomain {
					return fmt.Errorf("host type should not be %d", n.Typ)
				}
				if n.Net != 0 {
					return fmt.Errorf("net should not be %d", n.Net)
				}
				if n.String() != "0000000000000000000000000000000000000000000000000000000000000000@vite.org" {
					return fmt.Errorf("wrong string")
				}
				return nil
			},
		},
		{
			"0000000000000000000000000000000000000000000000000000000000000000@vite.org",
			func(n *Node, err error) error {
				if err != nil {
					return errors.New("error should be nil")
				}
				if n.ID != ZERO {
					return fmt.Errorf("id should be zero, not %s", n.ID.String())
				}
				if n.EndPoint.String() != "vite.org:8483" {
					return fmt.Errorf("endpoint should not be %s", n.EndPoint.String())
				}
				if n.Typ != HostDomain {
					return fmt.Errorf("host type should not be %d", n.Typ)
				}
				if n.Net != 0 {
					return fmt.Errorf("net should not be %d", n.Net)
				}
				if n.String() != "0000000000000000000000000000000000000000000000000000000000000000@vite.org" {
					return fmt.Errorf("wrong string")
				}
				return nil
			},
		},
		{
			"0000000000000000000000000000000000000000000000000000000000000001@127.0.0.1/2",
			func(n *Node, err error) error {
				if err != nil {
					return errors.New("error should be nil")
				}
				var id NodeID
				id[len(id)-1] = 1
				if n.ID != id {
					return fmt.Errorf("id should not be %s", n.ID.String())
				}
				if n.EndPoint.String() != "127.0.0.1:8483" {
					return fmt.Errorf("endpoint should not be %s", n.EndPoint.String())
				}
				if n.Typ != HostIPv4 {
					return fmt.Errorf("host type should not be %d", n.Typ)
				}
				if n.Net != 2 {
					return fmt.Errorf("net should not be %d", n.Net)
				}
				if n.String() != "0000000000000000000000000000000000000000000000000000000000000001@127.0.0.1/2" {
					return fmt.Errorf("wrong string")
				}
				return nil
			},
		},
		{
			"0000000000000000000000000000000000000000000000000000000000000001@127.0.0.1:8484/2",
			func(n *Node, err error) error {
				if err != nil {
					return errors.New("error should be nil")
				}
				var id NodeID
				id[len(id)-1] = 1
				if n.ID != id {
					return fmt.Errorf("id should not be %s", n.ID.String())
				}
				if n.EndPoint.String() != "127.0.0.1:8484" {
					return fmt.Errorf("endpoint should not be %s", n.EndPoint.String())
				}
				if n.Typ != HostIPv4 {
					return fmt.Errorf("host type should not be %d", n.Typ)
				}
				if n.Net != 2 {
					return fmt.Errorf("net should not be %d", n.Net)
				}
				if n.String() != "0000000000000000000000000000000000000000000000000000000000000001@127.0.0.1:8484/2" {
					return fmt.Errorf("wrong string")
				}
				return nil
			},
		},
		{
			"0000000000000000000000000000000000000000000000000000000000000001@[::1]/4",
			func(n *Node, err error) error {
				if err != nil {
					return errors.New("error should be nil")
				}
				var id NodeID
				id[len(id)-1] = 1
				if n.ID != id {
					return fmt.Errorf("id should not be %s", n.ID.String())
				}
				if n.EndPoint.String() != "[::1]:8483" {
					return fmt.Errorf("endpoint should not be %s", n.EndPoint.String())
				}
				if n.Typ != HostIPv6 {
					return fmt.Errorf("host type should not be %d", n.Typ)
				}
				if n.Net != 4 {
					return fmt.Errorf("net should not be %d", n.Net)
				}
				if n.String() != "0000000000000000000000000000000000000000000000000000000000000001@[::1]/4" {
					return fmt.Errorf("wrong string")
				}
				return nil
			},
		},
	}

	for _, samp := range samples {
		n, err := ParseNode(samp.url)
		if err = samp.equal(n, err); err != nil {
			t.Error(err)
		}
	}
}
