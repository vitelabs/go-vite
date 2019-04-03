/*
 * Copyright 2019 The go-vite Authors
 * This file is part of the go-vite library.
 *
 * The go-vite library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The go-vite library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with the go-vite library. If not, see <http://www.gnu.org/licenses/>.
 */

package discovery

import (
	"sort"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/common/math"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

const bucketSize = 32
const bucketNum = 32
const minDistance = vnode.IDBits - bucketNum

type nodeCollector interface {
	reset()
	bubble(id vnode.NodeID) bool
	// bucket cannot avoid duplicate node, table could
	add(node *Node) (toCheck *Node)
	remove(id vnode.NodeID) (n *Node)
	nodes(count int) (nodes []*Node)
	resolve(id vnode.NodeID) *Node
	size() int
}

type bucket interface {
	nodeCollector
	iterate(fn func(*Node)) // for test
	oldest() *Node
}

type Observer interface {
	Receive(n *vnode.Node)
	Sub(Subscriber)
	UnSub(Subscriber)
}

type Subscriber interface {
	Sub(observer Observer) (subId int)
	UnSub(subId int)
}

type nodeStore interface {
	Store(node *Node) (err error)
}

type nodeTable interface {
	nodeCollector
	addNodes(nodes []*Node)
	oldest() []*Node
	Sub(observer Observer) (subId int)
	UnSub(subId int)
	findNeighbors(id vnode.NodeID, count int) []*Node
	store(db nodeStore)
	resolveAddr(address string) *Node
}

type element struct {
	*Node
	next *element
}

// bucket no need possess chain lock
// because we operate bucket through table, so use table lock is more suited
type listBucket struct {
	// node-list, first item is an nil-element as head
	head *element

	tail *element

	// cap is the max number of nodes can stay in list, the same with sandby
	cap int

	// size is the current number of nodes in list
	_size int
}

func newListBucket(max int) bucket {
	e := &element{
		next: nil,
	}

	return &listBucket{
		head: e,
		tail: e,
		cap:  max,
	}
}

func (b *listBucket) iterate(fn func(*Node)) {
	for c := b.head.next; c != nil; c = c.next {
		fn(c.Node)
	}
}

func (b *listBucket) reset() {
	b._size = 0
	b.head.next = nil
	b.tail = b.head
}

// move the specific node to tail
func (b *listBucket) bubble(id vnode.NodeID) bool {
	if b._size == 0 {
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
			current.activeAt = time.Now()

			b.tail.next = current
			b.tail = current
			return true
		}
	}

	return false
}

// if bucket is not full, add node at tail, return nil
// return the first item, wait to ping-pong checked
func (b *listBucket) add(n *Node) (toCheck *Node) {
	// bucket is not full, add to tail
	if b._size < b.cap {
		n.addAt = time.Now()

		e := &element{
			Node: n,
			next: nil,
		}
		b.tail.next = e
		b.tail = e
		b._size++
		return
	}

	return b.oldest()
}

func (b *listBucket) remove(id vnode.NodeID) (n *Node) {
	for prev, current := b.head, b.head.next; current != nil; prev, current = current, current.next {
		if current.ID == id {
			n = current.Node

			prev.next = current.next
			if b.tail == current {
				b.tail = prev
			}

			b._size--
			return
		}
	}

	return nil
}

// retrieve count nodes from tail to head, if nodes is not enough, then retrieve them all
// if count == 0, retrieve all nodes
func (b *listBucket) nodes(count int) (nodes []*Node) {
	start := 0

	if count == 0 || b._size < count {
		nodes = make([]*Node, 0, b._size)
	} else {
		start = b._size - count
		nodes = make([]*Node, 0, count)
	}

	for i, current := 0, b.head.next; current != nil; i, current = i+1, current.next {
		if i >= start {
			nodes = append(nodes, current.Node)
		}
	}

	return
}

func (b *listBucket) oldest() *Node {
	if e := b.head.next; e != nil {
		return e.Node
	}

	return nil
}

func (b *listBucket) resolve(id vnode.NodeID) *Node {
	for current := b.head.next; current != nil; current = current.next {
		if current.ID == id {
			return current.Node
		}
	}

	return nil
}

func (b *listBucket) size() int {
	return b._size
}

type pinger interface {
	ping(n *Node) error
}

type table struct {
	rw sync.RWMutex

	buckets []bucket
	nodeMap map[string]*Node // key is address

	bucketFact func(capp int) bucket

	self *vnode.Node

	subId     int
	observers map[int]Observer

	socket pinger
}

func newTable(self *vnode.Node, bktSize, bktCount int, fact func(bktSize int) bucket, socket pinger) nodeTable {
	tab := &table{
		self:       self,
		buckets:    make([]bucket, bktCount),
		nodeMap:    make(map[string]*Node),
		bucketFact: fact,
		observers:  make(map[int]Observer),
		socket:     socket,
	}

	for i := range tab.buckets {
		tab.buckets[i] = tab.bucketFact(bktSize)
	}

	return tab
}

// reset operation clear all buckets
// will NOT clear observers
func (tab *table) reset() {
	tab.rw.Lock()
	defer tab.rw.Unlock()

	for _, bkt := range tab.buckets {
		bkt.reset()
	}

	tab.nodeMap = make(map[string]*Node)
}

// nodes retrieve count nodes from near to far
func (tab *table) nodes(count int) (nodes []*Node) {
	tab.rw.RLock()
	defer tab.rw.RUnlock()

	for _, bkt := range tab.buckets {
		ns := bkt.nodes(count)
		nodes = append(nodes, ns...)
		count -= len(ns)

		if count > 0 {
			continue
		}
	}

	return
}

func (tab *table) Sub(observer Observer) (subId int) {
	tab.rw.Lock()
	defer tab.rw.Unlock()

	tab.observers[tab.subId] = observer
	subId = tab.subId
	tab.subId++

	return
}

func (tab *table) UnSub(subId int) {
	tab.rw.Lock()
	defer tab.rw.Unlock()

	delete(tab.observers, subId)
}

func (tab *table) add(node *Node) (toCheck *Node) {
	if node == nil {
		return nil
	}

	go tab.notify(&node.Node)

	addr := node.Address()

	tab.rw.Lock()
	defer tab.rw.Unlock()

	// exist in table
	if _, ok := tab.nodeMap[addr]; ok {
		tab.rw.Unlock()
		return nil
	}

	bkt := tab.getBucket(node.ID)
	toCheck = bkt.add(node)
	if toCheck == nil {
		// bucket not full
		tab.nodeMap[addr] = node
		return
	}

	// ping oldNode
	go func() {
		err := tab.socket.ping(toCheck)
		if err != nil {
			tab.rw.Lock()
			tab.removeLocked(toCheck.ID)
			if bkt.add(node) == nil {
				tab.nodeMap[addr] = node
			}
			tab.rw.Unlock()
		}
	}()

	return
}

func (tab *table) addNodes(nodes []*Node) {
	tab.rw.Lock()
	defer tab.rw.Unlock()

	for _, n := range nodes {
		if n == nil {
			continue
		}

		addr := n.Address()
		if _, ok := tab.nodeMap[addr]; ok {
			continue
		}

		bkt := tab.getBucket(n.ID)

		if bkt.add(n) != nil {
			tab.nodeMap[addr] = n
		}
	}
}

func (tab *table) getBucket(id vnode.NodeID) bucket {
	d := vnode.Distance(tab.self.ID, id)
	if d <= minDistance {
		return tab.buckets[0]
	}
	return tab.buckets[d-minDistance-1]
}

func (tab *table) remove(id vnode.NodeID) (node *Node) {

	tab.rw.Lock()
	defer tab.rw.Unlock()

	return tab.removeLocked(id)
}

func (tab *table) removeLocked(id vnode.NodeID) (node *Node) {
	bkt := tab.getBucket(id)

	if node = bkt.remove(id); node != nil {
		addr := node.Address()
		delete(tab.nodeMap, addr)
	}

	return
}

func (tab *table) bubble(id vnode.NodeID) bool {
	bkt := tab.getBucket(id)

	tab.rw.Lock()
	defer tab.rw.Unlock()

	return bkt.bubble(id)
}

func (tab *table) findNeighbors(target vnode.NodeID, count int) []*Node {
	nes := closet{
		nodes: make([]*Node, 0, count),
		pivot: target,
	}

	tab.rw.RLock()
	defer tab.rw.RUnlock()

	for _, bkt := range tab.buckets {
		bkt.iterate(func(node *Node) {
			nes.push(node)
		})
	}

	return nes.nodes
}

func (tab *table) oldest() (nodes []*Node) {
	now := time.Now()

	tab.rw.RLock()
	defer tab.rw.RUnlock()

	for _, bkt := range tab.buckets {
		if n := bkt.oldest(); n != nil {
			if now.Sub(n.activeAt) > checkExpiration {
				nodes = append(nodes, n)
			}
		}
	}

	return
}

func (tab *table) notify(n *vnode.Node) {
	for _, ob := range tab.observers {
		ob.Receive(n)
	}
}

func (tab *table) size() int {
	tab.rw.RLock()
	defer tab.rw.RUnlock()

	count := 0
	for _, bkt := range tab.buckets {
		count += bkt.size()
	}

	return count
}

func (tab *table) resolve(id vnode.NodeID) *Node {
	bkt := tab.getBucket(id)

	tab.rw.Lock()
	defer tab.rw.Unlock()

	return bkt.resolve(id)
}

func (tab *table) resolveAddr(address string) *Node {
	tab.rw.RLock()
	defer tab.rw.RUnlock()

	return tab.nodeMap[address]
}

func (tab *table) store(db nodeStore) {
	now := time.Now()

	nodes := tab.nodes(math.MaxInt64)

	for _, n := range nodes {
		if now.Sub(n.addAt) > stayInTable {
			_ = db.Store(n)
		}
	}
}

// closet around the pivot
type closet struct {
	nodes []*Node
	pivot vnode.NodeID
}

func (c *closet) push(n *Node) {
	if n == nil {
		return
	}

	length := len(c.nodes)

	// sort.Search may return the index out of range
	dist := vnode.Distance(c.pivot, n.ID)
	further := sort.Search(length, func(i int) bool {
		return vnode.Distance(c.pivot, c.nodes[i].ID) > dist
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
