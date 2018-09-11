package discovery

import (
	crand "crypto/rand"
	"encoding/binary"
	"github.com/vitelabs/go-vite/log15"
	mrand "math/rand"
	"sort"
	"sync"
	"time"
)

var discvLog = log15.New("module", "p2p/discv")

// @section Bucket

const K = 16
const N = 17
const alpha = 3
const minDistance = 239
const maxFindFails = 5

// there is no need bucket have a lock,
// because we operate bucket through table, so table lock is feasible
type bucket struct {
	list []*Node
	cap  int
}

func newBucket(cap int) *bucket {
	if cap == 0 {
		cap = 16
	}

	return &bucket{
		list: make([]*Node, 0, cap),
		cap:  cap,
	}
}

// if node exists in bucket, then move node to tail, return nil
// if bucket is not full, add node at tail, return nil
// return the head item, wait to ping-pong checked
func (b *bucket) add(node *Node) (toCheck *Node) {
	if node == nil {
		return
	}

	if b.update(node) {
		return
	}

	// bucket is not full
	if len(b.list) < b.cap {
		b.list = append(b.list, node)
		return
	}

	for i, n := range b.list {
		// move to tail
		if n.ID == node.ID {
			copy(b.list[i:], b.list[i+1:])
			b.list[len(b.list)-1] = node
			return
		}
	}

	return b.list[0]
}

// move the node whose NodeID is id to tail
func (b *bucket) bubble(id NodeID) bool {
	for i, node := range b.list {
		if node.ID == id {
			copy(b.list[i:], b.list[i+1:])
			b.list[len(b.list)-1] = node
			return true
		}
	}
	return false
}

func (b *bucket) bubbleNode(node *Node) bool {
	return b.bubble(node.ID)
}

func (b *bucket) update(node *Node) bool {
	for i, n := range b.list {
		if n.ID == node.ID {
			b.list[i] = node
			return true
		}
	}

	return false
}

func (b *bucket) replace(old, new *Node) {
	for i, n := range b.list {
		if n.ID == old.ID {
			b.list[i] = new
			return
		}
	}
}

func (b *bucket) replaceAt(new *Node, i int) {
	b.list[i] = new
}

func (b *bucket) oldest() *Node {
	if len(b.list) > 0 {
		return b.list[0]
	}
	return nil
}

func (b *bucket) removeNode(node *Node) {
	b.remove(node.ID)
}

func (b *bucket) remove(id NodeID) {
	for i, n := range b.list {
		if n.ID == id {
			copy(b.list[i:], b.list[i+1:])
			b.list = b.list[:len(b.list)-1]
		}
	}
}

func (b *bucket) nodes() []*Node {
	return b.list
}

func (b *bucket) contains(node *Node) bool {
	for _, n := range b.list {
		if n.ID == node.ID {
			return true
		}
	}

	return false
}

// @section table
const seedCount = 20
const seedMaxAge = 7 * 24 * time.Hour
const minPingInterval = 3 * time.Minute

type table struct {
	lock      sync.RWMutex
	buckets   [N]*bucket
	bootNodes []*Node
	self      NodeID
	rand      *mrand.Rand
}

func newTable(self NodeID, bootNodes []*Node) *table {
	tab := &table{
		self:      self,
		bootNodes: bootNodes,
		rand:      mrand.New(mrand.NewSource(0)),
	}

	// init buckets
	for i, _ := range tab.buckets {
		tab.buckets[i] = &bucket{}
	}

	tab.resetRand()

	return tab
}

func (tab *table) resetRand() {
	var b [8]byte
	crand.Read(b[:])

	tab.lock.Lock()
	tab.rand.Seed(int64(binary.BigEndian.Uint64(b[:])))
	tab.lock.Unlock()
}

func (tab *table) randomNodes(dest []*Node) (count int) {
	tab.lock.Lock()
	defer tab.lock.Unlock()

	var allNodes [][]*Node
	for _, b := range tab.buckets {
		if b.length > 0 {
			allNodes = append(allNodes, b.list[:b.length])
		}
	}

	if len(allNodes) == 0 {
		return 0
	}

	// shuffle
	for i := 0; i < len(allNodes); i++ {
		j := tab.rand.Intn(len(allNodes))
		allNodes[i], allNodes[j] = allNodes[j], allNodes[i]
	}

	for j := 0; count < len(dest); j = (j + 1) % len(allNodes) {
		b := allNodes[j]
		dest[count] = b[len(b)-1]
		count++

		if len(b) == 1 {
			allNodes = append(allNodes[:j], allNodes[j+1:]...)
		} else {
			allNodes[j] = b[:len(b)-1]
		}
		if len(allNodes) == 0 {
			break
		}
	}

	return
}

func (tab *table) addNode(node *Node) {
	if node == nil {
		return
	}
	if node.ID == tab.self {
		return
	}

	bucket := tab.getBucket(node.ID)
	// todo check
	bucket.add(node)
}

func (tab *table) replaceNode(old, new *Node) {

}

func (tab *table) addNodes(nodes []*Node) {
	for _, n := range nodes {
		tab.addNode(n)
	}
}

func (tab *table) getBucket(id NodeID) *bucket {
	d := calcDistance(tab.self, id)
	if d <= minDistance {
		return tab.buckets[0]
	}
	return tab.buckets[d-minDistance-1]
}

func (tab *table) delete(node *Node) {
	tab.lock.Lock()
	defer tab.lock.Unlock()

	bucket := tab.getBucket(node.ID)
	bucket.removeNode(node)
}

func (tab *table) bubble(node *Node) {
	bucket := tab.getBucket(node.ID)
	bucket.add(node)
}

func (tab *table) findNeighbors(target NodeID) []*Node {
	neighbors := &neighbors{pivot: target}

	tab.traverse(func(n *Node) {
		neighbors.push(n, K)
	})

	return neighbors.nodes
}

func (tab *table) traverse(fn func(*Node)) {
	for _, b := range tab.buckets {
		for _, n := range b.nodes() {
			fn(n)
		}
	}
}

func (tab *table) pickOldest() (n *Node) {
	now := time.Now()

	for i := range tab.rand.Perm(N) {
		b := tab.buckets[i]
		n = b.oldest()

		if n == nil {
			continue
		}

		if now.Sub(n.lastPing) < minPingInterval {
			continue
		}

		return
	}

	return
}

// @section closet
// closest nodes to the target NodeID
type neighbors struct {
	nodes []*Node
	pivot NodeID
}

func (c *neighbors) push(n *Node, count int) {
	if n == nil {
		return
	}

	length := len(c.nodes)
	// sort.Search may return the index out of range
	further := sort.Search(length, func(i int) bool {
		return disCmp(c.pivot, c.nodes[i].ID, n.ID) > 0
	})

	// closest Nodes list is full.
	if length >= count {
		// replace the further one.
		if further < length {
			c.nodes[further] = n
		}
	} else {
		// increase c.nodes length first.
		c.nodes = append(c.nodes, nil)
		// insert n to furtherNodeIndex
		copy(c.nodes[further+1:], c.nodes[further:])
		c.nodes[further] = n
	}
}
