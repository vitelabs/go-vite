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

	// node has been in bucket, update the node info
	if b.bubble(node.ID) {
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

func (b *bucket) node(id NodeID) (*Node, int) {
	for i, n := range b.list {
		if n.ID.Equal(id) {
			return n, i
		}
	}

	return nil, -1
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
const minPingInterval = 3 * time.Minute

type table struct {
	lock    sync.RWMutex
	buckets []*bucket
	self    NodeID
	rand    *mrand.Rand
}

func newTable(self NodeID, bucketCount int) *table {
	tab := &table{
		self: self,
		rand: mrand.New(mrand.NewSource(0)),
	}

	// init buckets
	if bucketCount == 0 {
		bucketCount = N
	}
	tab.buckets = make([]*bucket, bucketCount)
	for i, _ := range tab.buckets {
		tab.buckets[i] = newBucket(K)
	}

	tab.initRand()

	return tab
}

func (tab *table) refresh() {
	tab.lock.Lock()
	defer tab.lock.Unlock()

	tab.initRand()

	for i, _ := range tab.buckets {
		tab.buckets[i] = newBucket(K)
	}
}

func (tab *table) initRand() {
	var b [8]byte
	crand.Read(b[:])

	tab.lock.Lock()
	defer tab.lock.Unlock()

	tab.rand.Seed(int64(binary.BigEndian.Uint64(b[:])))
}

func (tab *table) randomNodes(dest []*Node) (count int) {
	tab.lock.Lock()
	defer tab.lock.Unlock()

	var allNodes [][]*Node
	for _, b := range tab.buckets {
		if len(b.list) > 0 {
			allNodes = append(allNodes, b.list)
		}
	}

	rows := len(allNodes)
	if rows == 0 {
		return 0
	}

	// shuffle
	for i := 0; i < rows; i++ {
		j := tab.rand.Intn(rows)
		allNodes[i], allNodes[j] = allNodes[j], allNodes[i]
	}

	for j := 0; count < len(dest); j = (j + 1) % len(allNodes) {
		b := allNodes[j]
		dest[count] = b[len(b)-1]
		count++

		if len(b) == 1 {
			// remove this slice
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

func (tab *table) addNode(node *Node) *Node {
	if node == nil {
		return nil
	}
	if node.ID.Equal(tab.self) {
		return nil
	}

	tab.lock.Lock()
	defer tab.lock.Unlock()

	node.addAt = time.Now()
	bucket := tab.getBucket(node.ID)
	return bucket.add(node)
}

// if bucket is full, then we nil ping-pong check the oldest node
func (tab *table) mustAddNode(node *Node, check func(*Node) bool) {
	toChecked := tab.addNode(node)
	if toChecked != nil && check != nil {
		// check the old node failed, then replace it
		if !check(toChecked) {
			tab.lock.Lock()
			defer tab.lock.Unlock()
			node.addAt = time.Now()
			tab.replaceNode(toChecked, node)
		}
	}
}

func (tab *table) replaceNode(old, new *Node) {
	if old == nil || new == nil {
		return
	}

	tab.lock.Lock()
	defer tab.lock.Unlock()

	new.addAt = time.Now()
	bucket := tab.getBucket(old.ID)
	bucket.replace(old, new)
}

func (tab *table) updateNode(node *Node) {
	if node == nil {
		return
	}

	tab.lock.Lock()
	defer tab.lock.Unlock()

	bucket := tab.getBucket(node.ID)
	old, index := bucket.node(node.ID)
	if old != nil {
		node.addAt = old.addAt
		bucket.replaceAt(node, index)
	}
}

func (tab *table) addNodes(nodes []*Node) {
	tab.lock.Lock()
	defer tab.lock.Unlock()

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

func (tab *table) removeNode(node *Node) {
	tab.lock.Lock()
	defer tab.lock.Unlock()

	bucket := tab.getBucket(node.ID)
	bucket.removeNode(node)
}

func (tab *table) remove(id NodeID) {
	tab.lock.Lock()
	defer tab.lock.Unlock()

	bucket := tab.getBucket(id)
	bucket.remove(id)
}

func (tab *table) bubble(node *Node) {
	tab.lock.Lock()
	defer tab.lock.Unlock()

	bucket := tab.getBucket(node.ID)
	bucket.bubble(node.ID)
}

func (tab *table) findNeighbors(target NodeID, count int) *neighbors {
	tab.lock.RLock()
	defer tab.lock.RUnlock()

	neighbors := newNeighbors(target, count)

	tab.traverse(func(n *Node) {
		neighbors.push(n)
	})

	return neighbors
}

func (tab *table) traverse(fn func(*Node)) {
	tab.lock.RLock()
	defer tab.lock.RUnlock()

	for _, b := range tab.buckets {
		for _, n := range b.list {
			fn(n)
		}
	}
}

func (tab *table) pickOldest() (n *Node) {
	tab.lock.RLock()
	defer tab.lock.RUnlock()

	now := time.Now()

	for i := range tab.rand.Perm(N) {
		n = tab.buckets[i].oldest()

		if n == nil || now.Sub(n.lastPing) < minPingInterval {
			continue
		}

		return
	}

	return
}

// @section neighbors
// neighbors around the pivot
type neighbors struct {
	nodes []*Node
	pivot NodeID
}

func newNeighbors(pivot NodeID, cap int) *neighbors {
	return &neighbors{
		nodes: make([]*Node, 0, cap),
		pivot: pivot,
	}
}

func (c *neighbors) push(n *Node) {
	if n == nil {
		return
	}

	length := len(c.nodes)

	// sort.Search may return the index out of range
	further := sort.Search(length, func(i int) bool {
		return disCmp(c.pivot, c.nodes[i].ID, n.ID) > 0
	})

	// closest Nodes list is full.
	if length >= cap(c.nodes) {
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
