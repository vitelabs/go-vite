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

package p2p

import (
	"bytes"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

var id = vnode.RandomNodeID()

func TestNodeDB_Store(t *testing.T) {
	mdb, err := newDB("", 1, id)
	if err != nil {
		panic(err)
	}

	node := &Node{
		Node:     *vnode.MockNode(false, true),
		checkAt:  time.Now().Add(time.Hour),
		activeAt: time.Now().Add(2 * time.Hour),
	}

	err = mdb.Store(node)
	if err != nil {
		panic(err)
	}

	node2, err := mdb.Retrieve(node.ID)
	if err != nil {
		t.Errorf("retrieve error: %v", err)
	}

	if node2.ID != node.ID {
		t.Error("diff id")
	}
	if node2.EndPoint.String() != node.EndPoint.String() {
		t.Error("diff endpoint")
	}
	if !bytes.Equal(node2.Ext, node.Ext) {
		t.Error("diff extension")
	}
	if node2.Net != node.Net {
		t.Error("diff net")
	}
	if node2.checkAt.Unix() != node.checkAt.Unix() {
		t.Error("diff checkAt")
	}
	if node2.activeAt.Unix() != node.activeAt.Unix() {
		t.Error("diff activeAt")
	}

}

func TestNodeDB_Clean(t *testing.T) {

}
