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

	if !bytes.Equal(n.Hostname, n2.Hostname) {
		return fmt.Errorf("different Host %v %v", n.Hostname, n2.Hostname)
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
	n := MockNode(true)
	data, err := n.Serialize()
	if err != nil {
		t.Error(err)
	}

	n2, err := Deserialize(data)
	if err != nil {
		t.Error(err)
	}

	if err = compare(n, n2, true); err != nil {
		t.Error(err)
	}
}

func TestNode_String(t *testing.T) {
	n := MockNode(false)

	str := n.String()

	n2, err := ParseNode(str)
	if err != nil {
		t.Error(err)
	}

	if err = compare(n, n2, false); err != nil {
		t.Error(err)
	}
}

func TestCommonBits(t *testing.T) {
	var a, b NodeID
	total := len(a)

	// change last byte
	b[total-1] = b[total-1] | 1

	if Distance(a, b) != 8*total-1 {
		t.Fail()
	}

	// reset
	b[total-1] = 0

	// change first byte
	b[0] = b[0] | 1
	if Distance(a, b) != 7 {
		t.Fail()
	}

	// reset
	b[0] = 0

	// change random byte
	bytIndex := rand.Intn(total)
	bitIndex := rand.Intn(8)
	b[bytIndex] = 1 << uint(bitIndex)
	if Distance(a, b) != 8*bytIndex+(8-bitIndex) {
		t.Fail()
	}
}
