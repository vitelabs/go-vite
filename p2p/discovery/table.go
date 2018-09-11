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

type bucket struct {
	lock   sync.RWMutex
	list   [K]*Node
	length int
}

// if n exists in b.nodes, move n to tail, return nil
// if b.nodes is not full, add n at tail, return nil
// return the head item, to ping-pong checked.
func (b *bucket) add(node *Node) (toCheck *Node) {
	if node == nil {
		return
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	if b.contains(node) {
		return
	}

	if b.length < K {
		b.list[b.length] = node
		b.length++
		return
	}

	for i := 0; i < b.length; i++ {
		n := b.list[i]
		// move to tail
		if n.ID == node.ID {
			for j := i; j < b.length-2; j++ {
				b.list[j] = b.list[j+1]
			}
			b.list[b.length-1] = node
			return
		}
	}

	return b.list[0]
}

func (b *bucket) replace(old, new *Node) {
	b.lock.Lock()
	defer b.lock.Unlock()

	var n *Node
	for i := 0; i < b.length; i++ {
		n = b.list[i]
		if n.ID == old.ID {
			b.list[i] = new
			return
		}
	}
}

// replace the oldest one with node
func (b *bucket) update(node *Node, oldest *Node) {
	if node == nil || oldest == nil {
		return
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	if b.oldest().ID != oldest.ID {
		return
	}

	b.list[b.length-1] = node
}

func (b *bucket) oldest() *Node {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.list[0]
}

func (b *bucket) remove(node *Node) {
	b.lock.Lock()
	defer b.lock.Unlock()

	var n *Node
	for i := 0; i < b.length; i++ {
		n = b.list[i]
		if n.ID == node.ID {
			for j := i; j < b.length-1; j++ {
				b.list[j] = b.list[j+1]
			}
			b.length--
			b.list[b.length] = nil
			break
		}
	}
}

func (b *bucket) nodes() []*Node {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.list[:b.length]
}

func (b *bucket) contains(node *Node) bool {
	b.lock.RLock()
	defer b.lock.RUnlock()

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
	mutex     sync.RWMutex
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

func (tb *table) resetRand() {
	var b [8]byte
	crand.Read(b[:])

	tb.mutex.Lock()
	tb.rand.Seed(int64(binary.BigEndian.Uint64(b[:])))
	tb.mutex.Unlock()
}

func (tb *table) randomNodes(dest []*Node) (count int) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	var allNodes [][]*Node
	for _, b := range tb.buckets {
		if b.length > 0 {
			allNodes = append(allNodes, b.list[:b.length])
		}
	}

	if len(allNodes) == 0 {
		return 0
	}

	// shuffle
	for i := 0; i < len(allNodes); i++ {
		j := tb.rand.Intn(len(allNodes))
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

func (tb *table) addNode(node *Node) {
	if node == nil {
		return
	}
	if node.ID == tb.self {
		return
	}

	bucket := tb.getBucket(node.ID)
	// todo check
	bucket.add(node)
}

func (tb *table) replaceNode(old, new *Node) {

}

func (tb *table) addNodes(nodes []*Node) {
	for _, n := range nodes {
		tb.addNode(n)
	}
}

func (tb *table) getBucket(id NodeID) *bucket {
	d := calcDistance(tb.self, id)
	if d <= minDistance {
		return tb.buckets[0]
	}
	return tb.buckets[d-minDistance-1]
}

func (tb *table) delete(node *Node) {
	bucket := tb.getBucket(node.ID)
	bucket.remove(node)
}

func (tb *table) bubble(node *Node) {
	bucket := tb.getBucket(node.ID)
	bucket.add(node)
}

func (tb *table) findNeighbors(target NodeID) []*Node {
	neighbors := &neighbors{pivot: target}

	tb.traverse(func(n *Node) {
		neighbors.push(n, K)
	})

	return neighbors.nodes
}

func (tb *table) traverse(fn func(*Node)) {
	for _, b := range tb.buckets {
		for _, n := range b.nodes() {
			fn(n)
		}
	}
}

func (tb *table) pickOldest() (n *Node) {
	now := time.Now()

	for i := range tb.rand.Perm(N) {
		b := tb.buckets[i]
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
