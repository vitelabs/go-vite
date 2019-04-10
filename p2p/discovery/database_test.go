package discovery

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
