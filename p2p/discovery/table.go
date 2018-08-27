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
const Candidates = 10
const minDistance = 239

type bucket struct {
	nodes      []*Node
	candidates []*Node
}

func NewBucket() *bucket {
	return &bucket{
		nodes:      make([]*Node, 0, K),
		candidates: make([]*Node, 0, Candidates),
	}
}

// if n exists in b.nodes, move n to head, return true.
// if b.nodes is not full, set n to the first item, return true.
// if consider n as a candidate, then unshift n to b.candidates.
// return false.
func (b *bucket) check(n *Node) bool {
	if n == nil {
		return false
	}

	var i = 0
	for i, node := range b.nodes {
		if node == nil {
			break
		}
		// if n exists in b, then move n to head.
		if node.ID == n.ID {
			for j := i; j > 0; j-- {
				b.nodes[j] = b.nodes[j-1]
			}
			b.nodes[0] = n
			return true
		}
	}

	// if nodes is not full, set n to the first item
	if i < cap(b.nodes) {
		b.nodes = unshiftNode(b.nodes, n)
		b.candidates = removeNode(b.candidates, n)
		return true
	}

	return false
}
func (b *bucket) checkOrCandidate(n *Node) {
	used := b.check(n)
	if !used {
		b.candidates = unshiftNode(b.candidates, n)
	}
}

// obsolete the last node in b.nodes and return replacer.
func (b *bucket) obsolete(last *Node) *Node {
	if len(b.nodes) == 0 || b.nodes[len(b.nodes)-1].ID != last.ID {
		return nil
	}
	if len(b.candidates) == 0 {
		b.nodes = b.nodes[:len(b.nodes)-1]
		return nil
	}

	candidate := b.candidates[0]
	b.candidates = b.candidates[1:]
	b.nodes[len(b.nodes)-1] = candidate
	return candidate
}

func (b *bucket) remove(n *Node) {
	b.nodes = removeNode(b.nodes, n)
}

func removeNode(nodes []*Node, node *Node) []*Node {
	if node == nil {
		return nodes
	}

	for i, n := range nodes {
		if n != nil && n.ID == node.ID {
			return append(nodes[:i], nodes[i+1:]...)
		}
	}
	return nodes
}

// put node at first place of nodes without increase capacity, return the obsolete node.
func unshiftNode(nodes []*Node, node *Node) []*Node {
	if node == nil {
		return nil
	}

	// if node exist in nodes, then move to first.
	i := 0
	for _, n := range nodes {
		i++
		if n != nil && n.ID == node.ID {
			nodes[0], nodes[i] = nodes[i], nodes[0]
			return nodes
		}
	}

	// i equals len(nodes)
	if i < cap(nodes) {
		return append([]*Node{node}, nodes...)
	}

	// nodes is full, obsolete the last one.
	for i = i - 1; i > 0; i-- {
		nodes[i] = nodes[i-1]
	}
	nodes[0] = node
	return nodes
}

// @section table

const dbCurrentVersion = 1
const seedCount = 30
const alpha = 3

var refreshDuration = 30 * time.Minute
var storeDuration = 5 * time.Minute
var checkInterval = 3 * time.Minute
var watingTimeout = 2 * time.Minute // watingTimeout must be enough little, at least than checkInterval
var pingPongExpiration = 10 * time.Minute

type agent interface {
	ping(*Node) error
	findnode(NodeID, *Node) ([]*Node, error)
	close()
}

type table struct {
	mutex     sync.RWMutex
	buckets   [N]*bucket
	bootNodes []*Node
	db        *nodeDB
	agent     agent
	self      *Node
	rand      *mrand.Rand
	stopped   chan struct{}
	log       log15.Logger
}

func newTable(self *Node, agt agent, dbPath string, bootNodes []*Node) error {
	nodeDB, err := newDB(dbPath, dbCurrentVersion, self.ID)
	if err != nil {
		return err
	}

	tb := &table{
		self:      self,
		db:        nodeDB,
		agent:     agt,
		bootNodes: bootNodes,
		rand:      mrand.New(mrand.NewSource(0)),
		stopped:   make(chan struct{}),
		log:       log15.New("module", "discv/table"),
	}

	// init buckets
	for i, _ := range tb.buckets {
		tb.buckets[i] = NewBucket()
	}

	tb.resetRand()

	go tb.loop()

	return nil
}

func (tb *table) resetRand() {
	var b [8]byte
	crand.Read(b[:])

	tb.mutex.Lock()
	tb.rand.Seed(int64(binary.BigEndian.Uint64(b[:])))
	tb.mutex.Unlock()
}

func (tb *table) loadSeedNodes() {
	nodes := tb.db.randomNodes(seedCount)
	nodes = append(nodes, tb.bootNodes...)
	tb.log.Info("discover table load seed nodes", "size", len(nodes))

	for _, node := range nodes {
		tb.addNode(node)
	}
}

func (tb *table) readRandomNodes(ret []*Node) (n int) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	var allUsedNodes [][]*Node
	for _, b := range tb.buckets {
		if len(b.nodes) > 0 {
			allUsedNodes = append(allUsedNodes, b.nodes[:])
		}
	}

	if len(allUsedNodes) == 0 {
		return 0
	}

	for i := 0; i < len(allUsedNodes); i++ {
		j := tb.rand.Intn(len(allUsedNodes))
		allUsedNodes[i], allUsedNodes[j] = allUsedNodes[j], allUsedNodes[i]
	}

	for n, j := 0, 0; n < len(ret); n, j = n+1, (j+1)%len(allUsedNodes) {
		b := allUsedNodes[j]
		ret[n] = &(*b[0])
		if len(b) == 1 {
			allUsedNodes = append(allUsedNodes[:j], allUsedNodes[j+1:]...)
		} else {
			allUsedNodes[j] = b[1:]
		}
		if len(allUsedNodes) == 0 {
			break
		}
	}

	return n + 1
}

func (tb *table) loop() {
	checkTicker := time.NewTicker(checkInterval)
	refreshTimer := time.NewTimer(refreshDuration)
	storeTimer := time.NewTimer(storeDuration)

	defer checkTicker.Stop()
	defer refreshTimer.Stop()
	defer storeTimer.Stop()

	refreshDone := make(chan struct{})
	storeDone := make(chan struct{})

	// initial refresh
	go tb.refresh(refreshDone)

loop:
	for {
		select {
		case <-refreshTimer.C:
			go tb.refresh(refreshDone)
		case <-refreshDone:
			refreshTimer.Reset(refreshDuration)

		case <-checkTicker.C:
			go tb.checkLastNode()

		case <-storeTimer.C:
			go tb.storeNodes(storeDone)
		case <-storeDone:
			storeTimer.Reset(storeDuration)
		case <-tb.stopped:
			break loop
		}
	}

	if tb.agent != nil {
		tb.agent.close()
	}

	tb.db.close()
}

func (tb *table) addNode(node *Node) {
	if node == nil {
		return
	}
	if node.ID == tb.self.ID {
		return
	}

	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	bucket := tb.getBucket(node.ID)
	bucket.checkOrCandidate(node)
}
func (tb *table) getBucket(id NodeID) *bucket {
	d := calcDistance(tb.self.ID, id)
	if d <= minDistance {
		return tb.buckets[0]
	}
	return tb.buckets[d-minDistance-1]
}

func (tb *table) delete(node *Node) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	bucket := tb.getBucket(node.ID)
	bucket.remove(node)
}

func (tb *table) checkLastNode() {
	last, bucket := tb.pickLastNode()
	if last == nil {
		return
	}

	err := tb.agent.ping(last)

	if err != nil {
		tb.log.Info("obsolete node", "ID", last.ID, "error", err)
		bucket.obsolete(last)
	} else {
		bucket.check(last)
	}
}

func (tb *table) pickLastNode() (*Node, *bucket) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	for _, b := range tb.buckets {
		if len(b.nodes) > 0 {
			last := b.nodes[len(b.nodes)-1]
			if time.Now().Sub(last.lastpong) > pingPongExpiration {
				return last, b
			}
		}
	}

	return nil, nil
}

func (tb *table) refresh(done chan struct{}) {
	tb.log.Info("discv table begin refresh")

	tb.loadSeedNodes()

	tb.lookup(tb.self.ID)

	for i := 0; i < 3; i++ {
		var target NodeID
		crand.Read(target[:])
		tb.lookup(target)
	}

	done <- struct{}{}
}

func (tb *table) storeNodes(done chan struct{}) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	for _, b := range tb.buckets {
		for _, n := range b.nodes {
			tb.db.updateNode(n)
		}
	}

	done <- struct{}{}
}

func (tb *table) stop() {
	tb.log.Info("discv table stop")
	close(tb.stopped)
}

func (tb *table) lookup(target NodeID) []*Node {
	var asked = make(map[NodeID]bool)
	var seen = make(map[NodeID]bool)
	var reply = make(chan []*Node, alpha)
	var queries = 0
	var result *closest

	for {
		tb.mutex.Lock()
		result = tb.closest(target, K)
		tb.mutex.Unlock()

		if len(result.nodes) > 0 {
			break
		}

		time.Sleep(3 * time.Second)
	}

	asked[tb.self.ID] = true

	for i := 0; i < len(result.nodes); i++ {
		n := result.nodes[i]
		if !asked[n.ID] {
			asked[n.ID] = true
			go tb.findnode(n, target, reply)
			queries++
			if queries >= alpha {
				// todo: optimize latter
				time.Sleep(3 * time.Second)
				queries = 0
			}
		}
	}

	for _, n := range <-reply {
		if n != nil && !seen[n.ID] {
			seen[n.ID] = true
			result.push(n, K)
		}
	}

	return result.nodes
}

func (tb *table) closest(target NodeID, count int) *closest {
	result := &closest{target: target}
	for _, b := range tb.buckets {
		for _, n := range b.nodes {
			result.push(n, count)
		}
	}
	return result
}

func (tb *table) findnode(n *Node, targetID NodeID, reply chan<- []*Node) {
	nodes, err := tb.agent.findnode(targetID, n)

	if err != nil || len(nodes) == 0 {
		tb.delete(n)
	}

	for _, n := range nodes {
		tb.addNode(n)
	}

	reply <- nodes
}

// @section closet
// closest nodes to the target NodeID
type closest struct {
	nodes  []*Node
	target NodeID
}

func (c *closest) push(n *Node, count int) {
	if n == nil {
		return
	}

	length := len(c.nodes)
	furtherNodeIndex := sort.Search(length, func(i int) bool {
		return disCmp(c.target, c.nodes[i].ID, n.ID) > 0
	})

	// closest Nodes list is full.
	if length >= count {
		// replace the further one.
		if furtherNodeIndex < length {
			c.nodes[furtherNodeIndex] = n
		}
	} else {
		// increase c.nodes length first.
		c.nodes = append(c.nodes, nil)
		// insert n to furtherNodeIndex
		copy(c.nodes[furtherNodeIndex+1:], c.nodes[furtherNodeIndex:])
		c.nodes[furtherNodeIndex] = n
	}
}
