package discovery

import (
	"sort"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/p2p/network"
)

// K is the default bucketSize
const K = 128

// N is the default number of buckets
const N = 16
const minDistance = 256 - N

type element struct {
	*Node
	next *element
}

// bucket no need possess a lock
// because we operate bucket through table, so use table lock is more suited
type bucket struct {
	head *element // contains an head item
	tail *element
	cap  int
	size int
}

func newBucket(cap int) *bucket {
	e := &element{
		Node: &Node{},
		next: nil,
	}
	return &bucket{
		head: e,
		tail: e,
		cap:  cap,
	}
}

func (b *bucket) reset() {
	b.size = 0
	b.head.next = nil
	b.tail.next = nil
}

// move the node whose NodeID is id to tail
func (b *bucket) bubble(id NodeID) bool {
	if b.size == 0 {
		return false
	}

	if b.tail.ID == id {
		b.tail.activeAt = time.Now()
		return true
	}

	for prev, current := b.head, b.head.next; current != nil; prev, current = current, current.next {
		if current.ID == id {
			prev.next = current.next
			current.next = nil
			b.tail.next = current
			current.activeAt = time.Now()
			return true
		}
	}

	return false
}

// if bucket is not full, add node at tail, return nil
// return the first item, wait to ping-pong checked
func (b *bucket) add(node *Node) (toCheck *Node) {
	if node == nil {
		return nil
	}

	// bucket is not full, add to tail
	if b.size < b.cap {
		e := &element{
			Node: node,
			next: nil,
		}
		b.tail.next = e
		b.tail = e
		b.size++
		return nil
	}

	return b.oldest()
}

func (b *bucket) remove(id NodeID) {
	for prev, current := b.head, b.head.next; current != nil; prev, current = current, current.next {
		if current.ID == id {
			prev.next = current.next
			if b.tail == current {
				b.tail = prev
			}

			b.size--
			return
		}
	}
}

func (b *bucket) nodes(count int) (nodes []*Node) {
	start := 0
	if b.size > count {
		start = b.size - count
		nodes = make([]*Node, 0, count)
	} else {
		nodes = make([]*Node, 0, b.size)
	}

	for i, current := 0, b.head.next; current != nil; i, current = i+1, current.next {
		if i >= start {
			nodes = append(nodes, current.Node)
		}
	}

	return
}

func (b *bucket) oldest() *Node {
	if e := b.head.next; e != nil {
		return e.Node
	}

	return nil
}

func (b *bucket) resolve(id NodeID) *Node {
	for current := b.head.next; current != nil; current = current.next {
		if current.ID == id {
			return current.Node
		}
	}

	return nil
}

type table struct {
	m       sync.Map // addr: node
	mu      sync.RWMutex
	buckets []*bucket
	id      NodeID
	netID   network.ID
	chm     sync.Map // ch: bool
}

func newTable(id NodeID, netID network.ID) *table {
	tab := &table{
		id:      id,
		netID:   netID,
		buckets: make([]*bucket, N),
	}

	for i := range tab.buckets {
		tab.buckets[i] = newBucket(K)
	}

	return tab
}

func (tab *table) addNode(node *Node) *Node {
	if node == nil {
		return nil
	}

	if node.ID.Equal(tab.id) {
		return nil
	}

	// different network
	if node.Net != 0 && tab.netID != 0 && node.Net != tab.netID {
		return nil
	}

	addr := node.UDPAddr().String()
	// exist in table
	if n, ok := tab.m.Load(addr); ok {
		n := n.(*Node)
		n.Update(node)
		return nil
	}

	node.addAt = time.Now()
	tab.m.Store(addr, node)

	bkt := tab.getBucket(node.ID)

	tab.mu.Lock()
	oldest := bkt.add(node)
	tab.mu.Unlock()

	if oldest == nil {
		near := tab.buckets[0] == bkt
		tab.notify(node, near)
	}

	return oldest
}

func (tab *table) getBucket(id NodeID) *bucket {
	d := distance(tab.id, id)
	if d <= minDistance {
		return tab.buckets[0]
	}
	return tab.buckets[d-minDistance-1]
}

func (tab *table) remove(node *Node) {
	tab.m.Delete(node.UDPAddr().String())
	bkt := tab.getBucket(node.ID)

	tab.mu.Lock()
	defer tab.mu.Unlock()

	bkt.remove(node.ID)
}

func (tab *table) bubble(id NodeID) bool {
	bkt := tab.getBucket(id)

	tab.mu.Lock()
	defer tab.mu.Unlock()

	return bkt.bubble(id)
}

func (tab *table) bubbleAddr(addr string) bool {
	if node, ok := tab.m.Load(addr); ok {
		id := node.(*Node).ID
		bkt := tab.getBucket(id)

		tab.mu.Lock()
		defer tab.mu.Unlock()

		return bkt.bubble(id)
	}

	return false
}

func (tab *table) bubbleOrAdd(node *Node) {
	if _, ok := tab.m.Load(node.UDPAddr().String()); ok {
		if tab.bubble(node.ID) {
			return
		}
	}

	tab.addNode(node)
}

func (tab *table) findNeighbors(target NodeID, count int) []*Node {
	n := neighbors{
		nodes: make([]*Node, 0, count),
		pivot: target,
	}

	tab.mu.RLock()
	defer tab.mu.RUnlock()

	for _, bkt := range tab.buckets {
		nodes := bkt.nodes(K)
		for _, node := range nodes {
			if node.ID != target {
				n.push(node)
			}
		}
	}

	return n.nodes
}

func (tab *table) pickOldest() (nodes []*Node) {
	now := time.Now()

	tab.mu.RLock()
	defer tab.mu.RUnlock()

	for _, bkt := range tab.buckets {
		if n := bkt.oldest(); n != nil {
			if now.Sub(n.activeAt) > time.Minute || now.Sub(n.lastPing) > 3*time.Minute {
				nodes = append(nodes, n)
			}
		}
	}

	return
}

func (tab *table) SubNodes(ch chan<- *Node, near bool) {
	tab.chm.Store(ch, near)
}

func (tab *table) UnSubNodes(ch chan<- *Node) {
	tab.chm.Delete(ch)
}

func (tab *table) notify(node *Node, near bool) {
	tab.chm.Range(func(key, value interface{}) bool {
		ch, n := key.(chan<- *Node), value.(bool)
		if n == near {
			select {
			case ch <- node:
			default:
			}
		}

		return true
	})
}

func (tab *table) notifyAll(node *Node) {
	tab.chm.Range(func(key, value interface{}) bool {
		ch := key.(chan<- *Node)
		select {
		case ch <- node:
		default:
		}

		return true
	})
}

func (tab *table) needMore() bool {
	total := tab.size()

	return total*3 < N*K
}

func (tab *table) size() int {
	tab.mu.RLock()
	defer tab.mu.RUnlock()

	count := 0
	for _, bkt := range tab.buckets {
		count += bkt.size
	}

	return count
}

func (tab *table) resolve(addr string) *Node {
	v, ok := tab.m.Load(addr)
	if ok {
		return v.(*Node)
	}
	return nil
}

func (tab *table) resolveById(id NodeID) *Node {
	bkt := tab.getBucket(id)

	tab.mu.Lock()
	defer tab.mu.Unlock()

	return bkt.resolve(id)
}

type tableDB interface {
	storeNode(n *Node)
}

func (tab *table) store(db tableDB) {
	now := time.Now()
	tab.m.Range(func(key, value interface{}) bool {
		if n := value.(*Node); now.Sub(n.addAt) > stayInTable {
			db.storeNode(n)
		}
		return true
	})
}

func (tab *table) near() (nodes []*Node) {
	tab.mu.RLock()
	for _, bkt := range tab.buckets {
		if bkt.size > 0 {
			nodes = bkt.nodes(bkt.size)
			break
		}
	}
	tab.mu.RUnlock()
	return
}

//neighbors around the pivot
type neighbors struct {
	nodes []*Node
	pivot NodeID
}

func (c *neighbors) push(n *Node) {
	if n == nil {
		return
	}

	length := len(c.nodes)

	// sort.Search may return the index out of range
	dist := distance(c.pivot, n.ID)
	further := sort.Search(length, func(i int) bool {
		return distance(c.pivot, c.nodes[i].ID) > dist
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
