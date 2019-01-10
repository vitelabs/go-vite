package discovery

import (
	"testing"
	"time"
)

func TestDB_Store_Node(t *testing.T) {
	db, err := newDB("", 3, NodeID{})
	if err != nil {
		t.Fatal(err)
	}

	node := mockNode(true)
	node.lastPing = time.Now()
	node.lastFind = time.Now().Add(2 * time.Second)
	node.activeAt = time.Now().Add(5 * time.Second)
	node.mark = 10001

	db.storeNode(node)

	node2 := db.retrieveNode(node.ID)
	equal := compare(node, node2, true)
	if !equal {
		t.Fail()
	}

	if node.activeAt.Unix() != node2.activeAt.Unix() {
		t.Fail()
	}
	if node.lastFind.Unix() != node2.lastFind.Unix() {
		t.Fail()
	}
	if node.lastPing.Unix() != node2.lastPing.Unix() {
		t.Fail()
	}
	if node.mark != node2.mark {
		t.Fail()
	}
}
