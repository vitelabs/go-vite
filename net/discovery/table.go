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
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/net/vnode"
)

const bucketSize = 64
const bucketNum = 32

type nodeCollector interface {
	reset()
	// bubble the specific node to the least active position, return true if bubble
	// return false if cannot find the node
	bubble(id vnode.NodeID) bool
	// add node to the last recent active position
	add(node *Node) (toCheck *Node)
	// remove and return the specific node, if cannot find in the collector, return nil
	remove(id vnode.NodeID) (n *Node)
	// nodes return count nodes from collection, if count == 0 or count > collector.size() will return all nodes
	// from least active to last active
	// table return nodes from near to faraway
	nodes(count int) (nodes []*Node)
	// resolve return the specific node, if cannot find in the collector, return nil
	resolve(id vnode.NodeID) *Node
	// size is the number of nodes in the collector
	size() int
	// iterate nodes in collector, exit iteration when fn return false
	iterate(fn func(*Node) bool)
	// max is the collector`s capacity
	max() int
}

// bucket keep nodes from the same subtree, mean that, nodes have the same distance to the target id
type bucket interface {
	nodeCollector

	// oldest return the least recent active node, maybe nil
	oldest() *Node

	// replace the specific node with n, return false if cannot find the it in the bucket
	replace(id vnode.NodeID, n *Node) (success bool)
}

// Subscriber can be subscribed by observer
type Subscriber interface {
	// Sub return the unique id
	Sub(receiver func(n *vnode.Node)) (subId int)
	// UnSub received the id returned by Sub
	UnSub(subId int)
}

// nodeStore is the node database
type nodeStore interface {
	// StoreNode node into database, err is not nil if serialized error or database error
	StoreNode(node *Node) (err error)
}

// nodeTable has many buckets sorted by distance to self.ID from near to faraway
type nodeTable interface {
	nodeCollector
	Subscriber

	// addNodes receive a batch of nodes, can reduce lock cost of lock
	addNodes(nodes []*Node)
	// oldest return least recent active nodes from every bucket
	oldest() []*Node
	// findNeighbors return count nodes near the id
	findNeighbors(id vnode.NodeID, count int) []*Node

	findSource(id vnode.NodeID, count int) []*Node
	// store will store all nodes to database
	store(db nodeStore)
	// resolveAddr find the node match `node.Address() == address`
	resolveAddr(address string) *Node
	// toFind return the sub-tree need more nodes
	// return distance from the sub-tree
	subTreeToFind() uint
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
	count int
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

func (b *listBucket) iterate(fn func(*Node) bool) {
	for c := b.head.next; c != nil; c = c.next {
		if !fn(c.Node) {
			break
		}
	}
}

func (b *listBucket) reset() {
	b.count = 0
	b.head.next = nil
	b.tail = b.head
}

func (b *listBucket) replace(id vnode.NodeID, n *Node) bool {
	for current := b.head.next; current != nil; current = current.next {
		if current.ID == id {
			current.Node = n
			return true
		}
	}

	return false
}

// move the specific node to tail
func (b *listBucket) bubble(id vnode.NodeID) bool {
	if b.count == 0 {
		return false
	}

	if b.tail.ID == id {
		b.tail.activeAt = time.Now().Unix()
		return true
	}

	for prev, current := b.head, b.head.next; current != nil; prev, current = current, current.next {
		if current.ID == id {
			prev.next = current.next
			current.next = nil
			current.activeAt = time.Now().Unix()

			b.tail.next = current
			b.tail = current
			return true
		}
	}

	return false
}

// if n has existed in bucket, then return it to check, because address maybe changed.
// if bucket is not full, add node at tail, return nil.
// return the first item, wait to ping-pong checked.
func (b *listBucket) add(n *Node) (toCheck *Node) {
	for current := b.head.next; current != nil; current = current.next {
		if current.ID == n.ID {
			return current.Node
		}
	}

	// bucket is not full, add to tail
	if b.count < b.cap {
		n.addAt = time.Now().Unix()

		e := &element{
			Node: n,
			next: nil,
		}
		b.tail.next = e
		b.tail = e
		b.count++
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

			b.count--
			return
		}
	}

	return nil
}

// retrieve count nodes from tail to head, if nodes is not enough, then retrieve them all
// if count == 0, retrieve all nodes
func (b *listBucket) nodes(count int) (nodes []*Node) {
	start := 0

	if count == 0 || b.count < count {
		nodes = make([]*Node, 0, b.count)
	} else {
		start = b.count - count
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
	return b.count
}

func (b *listBucket) max() int {
	return b.cap
}

type pinger interface {
	ping(n *Node, callback func(err error))
}

type table struct {
	rw sync.RWMutex

	bucketSize, bucketNum int
	minDistance           uint

	buckets []bucket
	nodeMap map[string]*Node // key is address

	bucketFact func(capp int) bucket

	id    vnode.NodeID
	netId int

	subId     int
	recievers map[int]func(node *vnode.Node)

	socket pinger
}

func newTable(id vnode.NodeID, netId int, fact func(bktSize int) bucket, socket pinger) *table {
	tab := &table{
		id:          id,
		netId:       netId,
		bucketSize:  bucketSize,
		bucketNum:   bucketNum,
		minDistance: vnode.IDBits - uint(bucketNum) + 1,
		buckets:     make([]bucket, bucketNum),
		nodeMap:     make(map[string]*Node),
		bucketFact:  fact,
		recievers:   make(map[int]func(node *vnode.Node)),
		socket:      socket,
	}

	for i := range tab.buckets {
		tab.buckets[i] = tab.bucketFact(bucketSize)
	}

	return tab
}

func (tab *table) getId() vnode.NodeID {
	return tab.id
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
	if count == 0 {
		count = tab.bucketNum * tab.bucketSize
	}

	tab.rw.RLock()
	defer tab.rw.RUnlock()

	ids := rand.Perm(tab.bucketNum)
	for _, idx := range ids {
		bkt := tab.buckets[idx]
		ns := bkt.nodes(count)
		nodes = append(nodes, ns...)
		count -= len(ns)

		if count <= 0 {
			break
		}
	}

	return
}

func (tab *table) Sub(rec func(node *vnode.Node)) (subId int) {
	tab.rw.Lock()
	defer tab.rw.Unlock()

	subId = tab.subId
	tab.subId++

	tab.recievers[subId] = rec

	return
}

func (tab *table) UnSub(subId int) {
	tab.rw.Lock()
	defer tab.rw.Unlock()

	delete(tab.recievers, subId)
}

func (tab *table) notify(n *vnode.Node) {
	tab.rw.RLock()
	defer tab.rw.RUnlock()

	for _, rec := range tab.recievers {
		rec(n)
	}
}

func (tab *table) add(node *Node) (toCheck *Node) {
	if node.ID == tab.id || node.ID == vnode.ZERO {
		return nil
	}

	go tab.notify(&node.Node)

	addr := node.Address()

	bkt := tab.getBucket(node.ID)
	old := bkt.resolve(node.ID)
	if old == nil {
		tab.rw.Lock()
		old = tab.nodeMap[addr]
		tab.rw.Unlock()
	}

	if old != nil {
		// nothing change
		if old.Equal(&node.Node) {
			tab.rw.Lock()
			bkt.bubble(node.ID)
			tab.rw.Unlock()
			return nil
		}

		if old.needCheck() {
			// same address, different info
			// check old node
			go tab.checkRemove(old)
		}

		return old
	}

	// no nodes with the same id and address, and net is the same
	tab.rw.Lock()
	defer tab.rw.Unlock()
	toCheck = bkt.add(node)
	if toCheck == nil {
		// bucket not full
		tab.nodeMap[addr] = node
		return
	}

	if toCheck.needCheck() {
		// toCheck has different address with node
		go tab.checkReplace(bkt, toCheck, node)
	}

	return
}

func (tab *table) checkRemove(node *Node) {
	tab.socket.ping(node, func(err error) {
		if err != nil {
			tab.remove(node.ID)
			discvLog.Warn("remove %s", node)
		}
	})
}

func (tab *table) checkReplace(bkt bucket, oldNode, newNode *Node) {
	tab.socket.ping(oldNode, func(err error) {
		if err != nil {
			tab.rw.Lock()
			if bkt.replace(oldNode.ID, newNode) {
				delete(tab.nodeMap, oldNode.Address())
				tab.nodeMap[newNode.Address()] = newNode
			}
			tab.rw.Unlock()
			discvLog.Warn("replace %s to %s", oldNode, newNode)
		}
	})
}

func (tab *table) addNodes(nodes []*Node) {
	for _, node := range nodes {
		tab.add(node)
	}
}

func (tab *table) getBucket(id vnode.NodeID) bucket {
	d := vnode.Distance(tab.id, id)
	if d < tab.minDistance {
		return tab.buckets[0]
	}
	return tab.buckets[d-tab.minDistance]
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

func (tab *table) findNeighbors(target vnode.NodeID, count int) (nodes []*Node) {
	ns := &closet{
		nodes: make([]*Node, 0, count),
		pivot: target,
	}

	tab.rw.RLock()
	defer tab.rw.RUnlock()

	for _, bkt := range tab.buckets {
		bkt.iterate(func(node *Node) bool {
			if node.ID == target {
				return true
			}
			ns.push(node)
			return true
		})
	}

	return ns.nodes
}

func (tab *table) findSource(target vnode.NodeID, count int) (nodes []*Node) {
	exit := false

	tab.rw.RLock()
	defer tab.rw.RUnlock()

	for _, bkt := range tab.buckets {
		bkt.iterate(func(node *Node) bool {
			if node.couldFind() {
				nodes = append(nodes, node)
			}
			if len(nodes) > count {
				exit = true
				return false
			}

			return true
		})

		if exit {
			break
		}
	}

	return
}

func (tab *table) oldest() (nodes []*Node) {
	now := time.Now().Unix()

	tab.rw.RLock()
	defer tab.rw.RUnlock()

	for _, bkt := range tab.buckets {
		if n := bkt.oldest(); n != nil {
			if now-n.activeAt > checkExpiration {
				nodes = append(nodes, n)
			}
		}
	}

	return
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

func (tab *table) max() int {
	return tab.bucketNum * tab.bucketSize
}

func (tab *table) resolve(id vnode.NodeID) *Node {
	bkt := tab.getBucket(id)

	tab.rw.RLock()
	defer tab.rw.RUnlock()

	return bkt.resolve(id)
}

func (tab *table) resolveAddr(address string) *Node {
	tab.rw.RLock()
	defer tab.rw.RUnlock()

	return tab.nodeMap[address]
}

func (tab *table) store(db nodeStore) {
	now := time.Now().Unix()

	nodes := tab.nodes(0)

	for _, n := range nodes {
		if now-n.addAt > stayInTable {
			_ = db.StoreNode(n)
		}
	}
}

func (tab *table) iterate(fn func(*Node) bool) {
	nodes := tab.nodes(0)
	for _, n := range nodes {
		if !fn(n) {
			break
		}
	}
}

// toFind return the sub-tree need more nodes
func (tab *table) subTreeToFind() uint {
	perm := rand.Perm(len(tab.buckets))

	tab.rw.RLock()
	defer tab.rw.RUnlock()

	for _, i := range perm {
		if tab.buckets[i].size() < tab.bucketSize/2 {
			return uint(i) + tab.minDistance
		}
	}

	return 0
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
