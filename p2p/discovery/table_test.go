package discovery

import (
	"errors"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

var self = vnode.MockNode(false, true)

type mockPinger struct {
	fail bool
}

func (mp *mockPinger) ping(n *Node) error {
	if mp.fail {
		return errors.New("mock error")
	}

	return nil
}

func TestTable_add(t *testing.T) {
	// add until the nearest bucket is full
	mp := &mockPinger{false}
	tab := newTable(vnode.ZERO, bucketSize, bucketNum, newListBucket, mp)
	var node *Node
	for i := 0; i < bucketSize; i++ {
		node = &Node{
			Node: vnode.Node{
				ID: vnode.RandFromDistance(tab.id, 100),
				EndPoint: vnode.EndPoint{
					Host: []byte{0, 0, 0, 0},
					Port: i,
					Typ:  vnode.HostIPv4,
				},
				Net: self.Net,
				Ext: nil,
			},
		}

		if tab.add(node) != nil {
			t.Fail()
		}
	}

	// add another node
	node = &Node{
		Node: vnode.Node{
			ID: vnode.RandFromDistance(tab.id, 100),
			EndPoint: vnode.EndPoint{
				Host: []byte{0, 0, 0, 0},
				Port: 33,
				Typ:  vnode.HostIPv4,
			},
			Net: self.Net,
			Ext: nil,
		},
	}

	var oldest *Node
	if tab.add(node) == nil {
		t.Error("should return the oldest node to check")
	}
	time.Sleep(100 * time.Millisecond)
	if oldest = tab.oldest()[0]; oldest.Port != 0 {
		t.Error("the check node should be the first node")
	}

	// check false
	mp.fail = true
	if tab.add(node) == nil {
		t.Error("should return the oldest node to check")
	}

	time.Sleep(100 * time.Millisecond)
	if oldest = tab.oldest()[0]; oldest.Port != node.Port {
		t.Errorf("the oldest node should not be: %s", oldest.String())
	}
}

func TestTable_add2(t *testing.T) {
	// add until the nearest bucket is full
	mp := &mockPinger{false}
	tab := newTable(vnode.ZERO, bucketSize, bucketNum, newListBucket, mp)

	tab.add(&Node{
		Node: vnode.Node{
			ID: vnode.ZERO,
		},
	})

	if tab.size() != 0 {
		t.Error("table should be empty")
	}

	var id = vnode.RandomNodeID()
	node := &Node{
		Node: vnode.Node{
			ID: id,
			EndPoint: vnode.EndPoint{
				Host: []byte{127, 0, 0, 1},
				Port: 8483,
				Typ:  vnode.HostIPv4,
			},
		},
	}

	tab.add(node)
	if tab.size() != 1 {
		t.Errorf("table size should not be %d", tab.size())
	}

	tab.add(node)
	if tab.size() != 1 {
		t.Errorf("table size should not be %d", tab.size())
	}

	nodes := tab.nodes(0)
	if len(nodes) != 1 {
		t.Errorf("should not have %d nodes", len(nodes))
	}
	if nodes[0] != node {
		t.Errorf("should not be %s", nodes[0].String())
	}
}

func TestTable_add3(t *testing.T) {
	// add until the nearest bucket is full
	mp := &mockPinger{false}
	tab := newTable(vnode.ZERO, bucketSize, bucketNum, newListBucket, mp)

	node := &Node{
		Node: vnode.Node{
			ID: vnode.RandomNodeID(),
			EndPoint: vnode.EndPoint{
				Host: []byte{127, 0, 0, 1},
				Port: 8483,
				Typ:  vnode.HostIPv4,
			},
		},
	}

	if tocheck := tab.add(node); tocheck != nil {
		t.Errorf("check node should not be %s", tocheck.String())
	}

	// same id different address
	node2 := &Node{
		Node: vnode.Node{
			ID: node.ID,
			EndPoint: vnode.EndPoint{
				Host: []byte{127, 0, 0, 1},
				Port: 8485,
				Typ:  vnode.HostIPv4,
			},
		},
	}

	// check old success, should keep old
	if tocheck := tab.add(node2); tocheck != node {
		if tocheck == nil {
			t.Error("check node should not be nil")
		} else {
			t.Errorf("check node should not be %s", tocheck.String())
		}
	} else {
		time.Sleep(100 * time.Millisecond)
		if str := tab.resolve(node.ID).String(); str != node.String() {
			t.Errorf("should not be %s", str)
		}

		if n := tab.resolveAddr(node.Address()); n != node {
			t.Error("error resolve by address")
		}
		if n := tab.resolveAddr(node2.Address()); n != nil {
			t.Errorf("should not be %s", n.String())
		}
	}

	// check old failed, should keep new
	mp.fail = true
	if tocheck := tab.add(node2); tocheck != node {
		if tocheck == nil {
			t.Error("check node should not be nil")
		} else {
			t.Errorf("check node should not be %s", tocheck.String())
		}
	} else {
		time.Sleep(100 * time.Millisecond)
		if str := tab.resolve(node.ID).String(); str != node2.String() {
			t.Errorf("should not be %s", str)
		}
		if n := tab.resolveAddr(node.Address()); n != nil {
			t.Errorf("%s should be removed", n.String())
		}
		if n := tab.resolveAddr(node2.Address()); n != node2 {
			if n != nil {
				t.Errorf("should not be %s", n.String())
			} else {
				t.Error("should not be nil")
			}
		}
	}
}

func TestTable_nodes(t *testing.T) {
	var id vnode.NodeID
	mp := &mockPinger{false}
	tab := newTable(id, bucketSize, bucketNum, newListBucket, mp)

	// one node per bucket
	for i := tab.minDistance; i <= vnode.IDBits; i++ {
		node := &Node{
			Node: vnode.Node{
				ID: vnode.RandFromDistance(id, i),
				EndPoint: vnode.EndPoint{
					Host: []byte{127, 0, 0, 1},
					Port: int(i),
					Typ:  vnode.HostIPv4,
				},
			},
		}

		tab.add(node)
	}

	nodes := tab.nodes(0)
	if len(nodes) != bucketNum {
		t.Errorf("nodes count should not be %d", len(nodes))
	}

	var d, d2 uint
	for _, node := range nodes {
		if d2 = vnode.Distance(id, node.ID); d2 < d {
			t.Errorf("should be sort from near to faraway")
		}
		d = d2
	}
}

func TestTable_getBucket(t *testing.T) {
	var id = vnode.RandomNodeID()
	tab := newTable(id, bucketSize, bucketNum, newListBucket, nil)

	for i := uint(0); i <= vnode.IDBits; i++ {
		id2 := vnode.RandFromDistance(id, i)
		bkt := tab.getBucket(id2)

		if i <= tab.minDistance {
			if bkt != tab.buckets[0] {
				t.Error("should be near bucket")
			}
		} else {
			if bkt == tab.buckets[0] {
				t.Errorf("distance %d should not be near bucket", i)
			}
		}
	}
}

func TestCloset(t *testing.T) {
	const total = 5

	var id = vnode.RandomNodeID()

	var cls = closet{
		nodes: make([]*Node, 0, total),
		pivot: id,
	}

	var node *Node
	for i := uint(total + 1); i > 0; i-- {
		node = &Node{
			Node: vnode.Node{
				ID: vnode.RandFromDistance(id, i),
			},
		}
		cls.push(node)
	}

	var d, d2 uint
	var set bool
	for _, node = range cls.nodes {
		d2 = vnode.Distance(id, node.ID)
		if set {
			if d2 < d {
				t.Error("not sort")
			}
		} else {
			d = d2
		}
	}
}
