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

package database

import (
	"bytes"
	"testing"

	"github.com/vitelabs/go-vite/net/vnode"
)

var id = vnode.RandomNodeID()

func TestNodeDB_Store(t *testing.T) {
	mdb, err := New("", 1, id)
	if err != nil {
		panic(err)
	}

	node := vnode.MockNode(false, true)

	err = mdb.StoreNode(node)
	if err != nil {
		panic(err)
	}

	node2, err := mdb.RetrieveNode(node.ID)
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
}
