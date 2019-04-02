package discovery

import (
	"fmt"
	"testing"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

func TestBuck_add(t *testing.T) {
	const total = 5
	bkt := newListBucket(total)

	var node, first *Node
	for i := 0; i < total; i++ {
		node = &Node{
			Node: *vnode.MockNode(false, false),
		}

		if bkt.add(node) != nil {
			t.Fail()
		}

		if first == nil {
			first = node
		}
	}

	if bkt.add(node) != first {
		t.Fail()
	}
}

func TestBuck_reset(t *testing.T) {
	const total = 5
	bkt := newListBucket(total)

	var node *Node
	for i := 0; i < total; i++ {
		node = &Node{
			Node: *vnode.MockNode(false, false),
		}

		if bkt.add(node) != nil {
			t.Fail()
		}
	}

	bkt.reset()

	if bkt.size() != 0 {
		t.Fail()
	}

	var nodes = make([]*Node, total)
	for i := 0; i < total; i++ {
		node = &Node{
			Node: *vnode.MockNode(false, false),
		}

		if bkt.add(node) != nil {
			t.Fail()
		}

		nodes[i] = node
	}

	var index = 0
	bkt.iterate(func(node *Node) {
		if node != nodes[index] {
			t.Fail()
		}
		index++
	})
}

func TestBuck_bubble(t *testing.T) {
	const total = 5
	bkt := newListBucket(total)

	var i int
	var node *Node
	var nodes = make([]*Node, total)
	for i = 0; i < total; i++ {
		node = &Node{
			Node: *vnode.MockNode(false, false),
		}

		if bkt.add(node) != nil {
			t.Fail()
		}

		nodes[i] = node
	}

	for _, node = range nodes {
		fmt.Println(node.ID)
	}

	for i, node = range nodes {
		node = nodes[i]
		if !bkt.bubble(node.ID) {
			t.Error("bubble false")
		}

		next := (i + 1) % total
		if bkt.oldest() != nodes[next] {
			t.Errorf("oldest is wrong: %s %s", bkt.oldest().ID, nodes[next].ID)
		}
	}
}

func TestBuck_remove(t *testing.T) {
	const total = 5
	bkt := newListBucket(total)

	var i int
	var node *Node
	var nodes = make([]*Node, total)
	for i = 0; i < total; i++ {
		node = &Node{
			Node: *vnode.MockNode(false, false),
		}

		if bkt.add(node) != nil {
			t.Fail()
		}

		nodes[i] = node
	}

	for i, node = range nodes {
		if n2 := bkt.remove(node.ID); n2 != node {
			t.Fail()

			if bkt.size() != total-i-1 {
				t.Fail()
			}

			var j = i
			bkt.iterate(func(node *Node) {
				if node != nodes[j] {
					t.Fail()
				}
				j++
			})
		}
	}

	if bkt.remove(vnode.ZERO) != nil {
		t.Fail()
	}

	// add again
	for i = 0; i < total; i++ {
		node = &Node{
			Node: *vnode.MockNode(false, false),
		}

		if bkt.add(node) != nil {
			t.Fail()
		}

		nodes[i] = node
	}

	i = 0
	bkt.iterate(func(node *Node) {
		if node != nodes[i] {
			t.Fail()
		}
		i++
	})
}

func TestBuck_nodes(t *testing.T) {
	const total = 5
	bkt := newListBucket(total)

	var i int
	var node *Node
	var nodes = make([]*Node, total)
	for i = 0; i < total; i++ {
		node = &Node{
			Node: *vnode.MockNode(false, false),
		}

		if bkt.add(node) != nil {
			t.Fail()
		}

		nodes[i] = node
	}

	//for _, node = range nodes {
	//	fmt.Println(node.ID)
	//}

	nodes2 := bkt.nodes(total / 2)
	if len(nodes2) != total/2 {
		t.Errorf("length wrong: %d %d", len(nodes2), total/2)
	}
	//fmt.Println("--------")
	//for _, node = range nodes2 {
	//	fmt.Println(node.ID)
	//}
	for i, node = range nodes2 {
		if node != nodes[total-len(nodes2)+i] {
			t.Errorf("different node: %s %s", node.ID, nodes[total-len(nodes2)+i].ID)
		}
	}

	nodes2 = bkt.nodes(total + 1)
	if len(nodes2) != total {
		t.Fail()
	}

	for i, node = range nodes2 {
		if node != nodes[i] {
			t.Fail()
		}
	}

	nodes2 = bkt.nodes(0)
	if len(nodes2) != total {
		t.Fail()
	}

	for i, node = range nodes2 {
		if node != nodes[i] {
			t.Fail()
		}
	}
}

func TestBuck_resolve(t *testing.T) {
	const total = 5
	bkt := newListBucket(total)

	var i int
	var node *Node
	var nodes = make([]*Node, total)
	for i = 0; i < total; i++ {
		node = &Node{
			Node: *vnode.MockNode(false, false),
		}

		if bkt.add(node) != nil {
			t.Fail()
		}

		nodes[i] = node
	}

	for _, node = range nodes {
		if bkt.resolve(node.ID) != node {
			t.Fail()
		}
	}

	if bkt.resolve(vnode.ZERO) != nil {
		t.Fail()
	}
}

func TestBuck_size(t *testing.T) {
	const total = 5
	bkt := newListBucket(total)
	if bkt.size() != 0 {
		t.Fail()
	}

	var i int
	var node *Node
	var nodes = make([]*Node, total)
	for i = 0; i < total; i++ {
		node = &Node{
			Node: *vnode.MockNode(false, false),
		}

		if bkt.add(node) != nil {
			t.Fail()
		}

		nodes[i] = node

		if bkt.size() != i+1 {
			t.Fail()
		}
	}

	for i, node = range nodes {
		bkt.remove(node.ID)
		if bkt.size() != total-1-i {
			t.Fail()
		}
	}
}

func TestBuck_oldest(t *testing.T) {
	const total = 5
	bkt := newListBucket(total)

	if bkt.oldest() != nil {
		t.Fail()
	}

	var i int
	var node *Node
	var nodes = make([]*Node, total)
	for i = 0; i < total; i++ {
		node = &Node{
			Node: *vnode.MockNode(false, false),
		}

		if bkt.add(node) != nil {
			t.Fail()
		}

		nodes[i] = node

		if bkt.size() != i+1 {
			t.Fail()
		}
	}

	for i, node = range nodes {
		bkt.bubble(node.ID)
		next := (i + 1) % total
		if bkt.oldest() != nodes[next] {
			t.Fail()
		}
	}
}
