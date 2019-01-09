package discovery

import (
	"crypto/rand"
	"fmt"
	"net"
	"testing"
)

func TestBucket_Add(t *testing.T) {
	bkt := newBucket(K)

	var node *Node
	var first *Node
	for i := 0; i < K; i++ {
		node = mockNode(true)
		if first == nil {
			first = node
		}
		bkt.add(node)
	}

	if bkt.size != K {
		t.Fail()
	}

	if bkt.tail.Node != node {
		t.Fail()
	}

	// overflow
	toCheck := bkt.add(node)
	if toCheck != first {
		t.Fail()
	}
	if bkt.size != K {
		t.Fail()
	}
}

func TestBucket_Remove(t *testing.T) {
	bkt := newBucket(K)
	var node *Node
	var nodes = make([]*Node, K)
	for i := 0; i < K; i++ {
		node = mockNode(true)
		nodes[i] = node
		bkt.add(node)
	}

	for i := K - 1; i > -1; i-- {
		bkt.remove(nodes[i].ID)
		if bkt.size != i {
			t.Fail()
		}
	}

	for i := 0; i < K; i++ {
		bkt.add(nodes[i])
		if bkt.size != i+1 {
			t.Fail()
		}
	}
}

func TestBucket_Bubble(t *testing.T) {
	bkt := newBucket(K)
	var id NodeID
	bol := bkt.bubble(id)
	if bol {
		t.Fail()
	}

	var old *Node
	var first *Node
	for i := 0; i < K; i++ {
		node := mockNode(false)
		if first == nil {
			first = node
		}
		old = bkt.add(node)
	}

	if old != nil {
		t.Fail()
	}

	old = bkt.add(mockNode(false))
	if old != first {
		t.Fail()
	}

	bol = bkt.bubble(first.ID)
	if !bol {
		t.Fail()
	}

	bkt.bubble(id)
}

func TestTable_Find(t *testing.T) {
	var id NodeID
	rand.Read(id[:])
	table := newTable(id, 0)
	port := uint16(8483)
	for i := 0; i < 10000; i++ {
		var id2 NodeID
		rand.Read(id2[:])

		table.addNode(&Node{
			ID:  id2,
			IP:  net.IPv4(127, 0, 0, 1),
			UDP: port,
		})
		port++
	}

	const want = 10
	nodes := table.findNeighbors(id, want)

	if len(nodes) != want {
		t.Fail()
	}
	for _, bkt := range table.buckets {
		fmt.Println(bkt.size)
	}
	fmt.Println("total", table.size())
}

func TestTable_Delete(t *testing.T) {
	var id NodeID
	rand.Read(id[:])
	table := newTable(id, 0)

	var id2 NodeID
	port := uint16(8483)
	for i := 0; i < 100; i++ {
		rand.Read(id2[:])

		n := &Node{
			ID:  id2,
			IP:  net.IPv4(127, 0, 0, 1),
			UDP: port,
		}
		table.addNode(n)

		port++
	}

	n := table.resolveById(id2)
	if n == nil || n.ID != id2 || n.UDP != (port-1) {
		fmt.Println(n, id2)
		t.Fail()
	}

	table.removeById(id2)
	n = table.resolveById(id2)
	if n != nil {
		t.Fail()
	}
}
