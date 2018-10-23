package discovery

import (
	crand "crypto/rand"
	"encoding/binary"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p/network"
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

type nodeList struct {
	node *Node
	next *nodeList
}

func (n *nodeList) tail() (item *nodeList) {
	for item = n; item.next != nil; item = item.next {
		// do nothing
	}
	return
}

// bucket no need possess a lock
// because we operate bucket through table, so use table lock is more suited
type bucket struct {
	list *nodeList // contains an head item
	cap  int
	size int
}

func newBucket(cap int) *bucket {
	if cap == 0 {
		cap = K
	}

	return &bucket{
		list: new(nodeList),
		cap:  cap,
	}
}

func (b *bucket) reset() {
	b.size = 0
	b.list.next = nil
}

// the last item
func (b *bucket) tail() (item *nodeList) {
	return b.list.tail()
}

// move the node whose NodeID is id to tail
func (b *bucket) bubble(id NodeID) bool {
	for prev, current := b.list, b.list.next; current != nil; prev, current = current, current.next {
		if current.node.ID == id {
			// move the target Item to tail
			for prev.next = current.next; prev.next != nil; prev = prev.next {
				// do nothing
			}
			current.next = nil
			prev.next = current
			return true
		}
	}

	return false
}

// if node exists in bucket, then move node to tail, return nil
// if bucket is not full, add node at tail, return nil
// return the first item, wait to ping-pong checked
func (b *bucket) add(node *Node) (toCheck *Node) {
	if node == nil {
		return
	}

	// node has been in bucket, update the node info
	if b.bubble(node.ID) {
		return
	}

	// bucket is not full, add to tail
	if b.size < b.cap {
		b.tail().next = &nodeList{
			node: node,
			next: nil,
		}
		b.size++
		return
	}

	return b.oldest()
}

func (b *bucket) bubbleNode(node *Node) bool {
	return b.bubble(node.ID)
}

func (b *bucket) replace(old, new *Node) {
	item := b.list
	for item.next != nil {
		if item.node.ID == old.ID {
			item.node = new
			return
		}
	}
}

func (b *bucket) oldest() *Node {
	first := b.list.next
	if first == nil {
		return nil
	}

	return first.node
}

func (b *bucket) removeNode(node *Node) {
	b.remove(node.ID)
}

func (b *bucket) remove(id NodeID) {
	prev, item := b.list, b.list.next
	for item != nil {
		if item.node.ID == id {
			prev.next = item.next
			b.size--
			return
		}
		prev, item = item, item.next
	}
}

func (b *bucket) node(id NodeID) *Node {
	item := b.list.next
	for item != nil {
		if item.node.ID == id {
			return item.node
		}
		item = item.next
	}

	return nil
}

func (b *bucket) contains(node *Node) bool {
	item := b.list.next
	for item != nil {
		if item.node.ID == node.ID {
			return true
		}
		item = item.next
	}

	return false
}

func (b *bucket) nodes() []*Node {
	if b.size == 0 {
		return nil
	}

	nodes := make([]*Node, b.size)
	for i, item := 0, b.list.next; item != nil; i, item = i+1, item.next {
		nodes[i] = item.node
	}

	return nodes
}

// @section table
const minPingInterval = 3 * time.Minute

type table struct {
	lock    sync.RWMutex
	buckets []*bucket
	self    NodeID
	rand    *mrand.Rand
	netId   network.ID
}

func newTable(self NodeID, netId network.ID) *table {
	tab := &table{
		self:  self,
		rand:  mrand.New(mrand.NewSource(0)),
		netId: netId,
	}

	// init buckets
	tab.buckets = make([]*bucket, N)
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
		tab.buckets[i].reset()
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
		if b.size > 0 {
			allNodes = append(allNodes, b.nodes())
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

func (tab *table) nodes() (nodes []*Node) {
	for _, b := range tab.buckets {
		if b.size > 0 {
			nodes = append(nodes, b.nodes()...)
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

	if node.Net != tab.netId {
		return nil
	}

	tab.lock.Lock()
	defer tab.lock.Unlock()

	node.addAt = time.Now()
	bucket := tab.getBucket(node.ID)
	return bucket.add(node)
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
		for _, n := range b.nodes() {
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

func (tab *table) Mark(id NodeID, lifetime int64) {
	tab.lock.Lock()
	defer tab.lock.Unlock()

	bucket := tab.getBucket(id)
	if n := bucket.node(id); n != nil {
		n.weight = lifetime
	}
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
