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
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p/vnode"
)

const seedMaxAge = 7 * 24 * time.Hour
const tRefresh = 24 * time.Hour        // refresh the node table at tRefresh intervals
const storeInterval = 15 * time.Minute // store nodes in table to db at storeDuration intervals
const checkInterval = 10 * time.Second // check the oldest node in table at checkInterval intervals
const checkExpiration = time.Hour      // should check again if last check is an hour ago
const stayInTable = 5 * time.Minute    // minimal duration node stay in table can be store in db
const dbCleanInterval = time.Hour

var errDiscoveryIsRunning = errors.New("discovery is running")
var errDiscoveryIsStopped = errors.New("discovery is stopped")
var errDifferentNet = errors.New("different net")
var errPingFailed = errors.New("failed to ping node")

var discvLog = log15.New("module", "discovery")

// Discovery is the interface to discovery other node
type Discovery interface {
	Start() error
	Stop() error
	Delete(id vnode.NodeID, reason error)
	GetNodes(count int) []*vnode.Node
	FindNodes(node *vnode.Node, count int) []*vnode.Node
	Resolve(id vnode.NodeID) *vnode.Node
	SetFinder(f Finder)
	AllNodes() []*vnode.Node
}

type db interface {
	Store(node *Node) (err error)
	ReadNodes(count int, maxAge time.Duration) []*Node
	Close() error
	Clean(expiration time.Duration)
	Remove(ID vnode.NodeID)
}

type discovery struct {
	*Config
	node *vnode.Node

	booters []booter

	table nodeTable

	finder Finder

	// nodes wait to check
	stage map[string]*Node // key is the communication address, not endpoint
	mu    sync.Mutex

	socket socket

	db db

	running int32
	term    chan struct{}

	looking int32 // is looking self

	refreshing bool
	cond       *sync.Cond

	wg sync.WaitGroup

	log log15.Logger
}

func (d *discovery) GetNodes(count int) []*vnode.Node {
	// can NOT use rw.RLock(), will panic
	d.mu.Lock()
	for d.refreshing {
		d.cond.Wait()
	}
	d.mu.Unlock()

	return d.finder.GetNodes(count)
}

func (d *discovery) FindNodes(node *vnode.Node, count int) []*vnode.Node {
	// todo
	return nil
}

func (d *discovery) SetFinder(f Finder) {
	if d.finder != nil {
		d.finder.UnSub(d.table)
	}

	d.finder = f
	d.finder.Sub(d.table)
}

func (d *discovery) Delete(id vnode.NodeID, reason error) {
	d.table.remove(id)
	d.db.Remove(id)

	d.log.Warn(fmt.Sprintf("remove node %s: %v", id.String(), reason))
}

func (d *discovery) Resolve(id vnode.NodeID) (ret *vnode.Node) {
	node := d.table.resolve(id)
	if node != nil {
		return &node.Node
	}

	if nodes := d.lookup(id, 1); len(nodes) > 0 {
		if nodes[0].ID == id {
			return &(nodes[0].Node)
		}
	}

	return
}

// New create a Discovery implementation
func New(cfg *Config) Discovery {
	d := &discovery{
		Config: cfg,
		stage:  make(map[string]*Node),
		log:    discvLog,
	}

	d.cond = sync.NewCond(&d.mu)

	d.node = cfg.Node()

	d.socket = newAgent(cfg.PrivateKey(), d.node, cfg.ListenAddress, d.handle)

	d.table = newTable(d.node.ID, d.NetID, cfg.BucketSize, cfg.BucketCount, newListBucket, d)

	d.SetFinder(&closetFinder{table: d.table})

	return d
}

func (d *discovery) Start() (err error) {
	if !atomic.CompareAndSwapInt32(&d.running, 0, 1) {
		return errDiscoveryIsRunning
	}

	// open database
	d.db, err = newDB(d.Config.DataDir, 3, d.node.ID)
	if err != nil {
		return fmt.Errorf("failed to create database: %v", err)
	}

	// retrieve boot nodes
	d.booters = append(d.booters, newDBBooter(d.db))
	if len(d.BootSeeds) > 0 {
		d.booters = append(d.booters, newNetBooter(d.node, d.BootSeeds))
	}
	if len(d.BootNodes) > 0 {
		var bt booter
		bt, err = newCfgBooter(d.BootNodes, d.node)
		if err != nil {
			return err
		}
		d.booters = append(d.booters, bt)
	}

	// open socket
	err = d.socket.start()
	if err != nil {
		return fmt.Errorf("failed to start udp server: %v", err)
	}

	d.term = make(chan struct{})

	// maintain node table
	d.wg.Add(1)
	go d.tableLoop()

	// find more nodes
	d.wg.Add(1)
	go d.findLoop()

	return
}

func (d *discovery) Stop() (err error) {
	if atomic.CompareAndSwapInt32(&d.running, 1, 0) {
		close(d.term)
		d.wg.Wait()

		err = d.socket.stop()
		err = d.db.Close()

		return
	}

	return errDiscoveryIsStopped
}

func (d *discovery) AllNodes() []*vnode.Node {
	nodes := d.table.nodes(0)
	vnodes := make([]*vnode.Node, len(nodes))
	for i, n := range nodes {
		vnodes[i] = &n.Node
	}

	return vnodes
}

// ping and update a node, will be blocked, so should invoked by goroutine
func (d *discovery) ping(n *Node) error {
	n.checking = true
	ch := make(chan *Node, 1)
	err := d.socket.ping(n, ch)

	if err != nil {
		n.checking = false
		return err
	}

	n2 := <-ch
	n.checking = false
	if n2 == nil {
		return errPingFailed
	}

	if n2.Net != d.node.Net {
		return errDifferentNet
	}

	n.checkAt = time.Now()
	n.update(n2)
	return nil
}

func (d *discovery) pingDelete(n *Node) {
	if d.ping(n) != nil {
		d.table.resolve(n.ID)
	}
}

func (d *discovery) tableLoop() {
	defer d.wg.Done()

	d.init()

	checkTicker := time.NewTicker(checkInterval)
	refreshTicker := time.NewTimer(tRefresh)
	storeTicker := time.NewTicker(storeInterval)
	dbTicker := time.NewTicker(dbCleanInterval)

	defer checkTicker.Stop()
	defer refreshTicker.Stop()
	defer storeTicker.Stop()
	defer dbTicker.Stop()

Loop:
	for {
		select {
		case <-d.term:
			break Loop
		case <-checkTicker.C:
			nodes := d.table.oldest()
			for _, node := range nodes {
				go d.pingDelete(node)
			}

		case <-refreshTicker.C:
			go d.init()

		case <-storeTicker.C:
			d.table.store(d.db)

		case <-dbTicker.C:
			d.db.Clean(seedMaxAge)
		}
	}

	d.table.store(d.db)
}

func (d *discovery) findLoop() {
	defer d.wg.Done()

	initduration := 10 * time.Second
	maxDuration := 640 * time.Second
	duration := initduration

	timer := time.NewTimer(duration)
	defer timer.Stop()

Loop:
	for {
		select {
		case <-d.term:
			break Loop

		case <-timer.C:
			if distance := d.table.toFind(); distance > 0 {
				d.findSubTree(distance)
			}

			if duration < maxDuration {
				duration *= 2
			} else {
				duration = initduration
			}

			timer.Reset(duration)
		}
	}
}

func (d *discovery) handle(pkt *packet) {
	defer recyclePacket(pkt)

	d.table.bubble(pkt.id)

	switch pkt.c {
	case codePing:
		n := nodeFromPing(pkt)
		if n.Net == d.node.Net {
			_ = d.socket.pong(pkt.hash, n)
		}

		d.table.bubble(n.ID)
		d.table.add(n)

	case codePong:
		// nothing
		// pong will be handle by requestPool

	case codeFindnode:
		find := pkt.body.(*findnode)
		nodes := d.table.findNeighbors(find.target, int(find.count))
		eps := make([]*vnode.EndPoint, len(nodes))
		for i, n := range nodes {
			eps[i] = &n.EndPoint
		}
		_ = d.socket.sendNodes(eps, pkt.from)

	case codeNeighbors:
		// nothing
	}
}

// receiveNode get from bootNodes or ping message, check and put into table
func (d *discovery) receiveNode(n *Node) error {
	if n.Net != d.node.Net {
		return errDifferentNet
	}

	if d.table.resolve(n.ID) != nil {
		return nil
	}

	if n.shouldCheck() {
		return d.checkNode(n)
	} else {
		d.table.add(n)
	}

	return nil
}

// receiveEndPoint from neighbors message
func (d *discovery) receiveEndPoint(e vnode.EndPoint) (n *Node, err error) {
	addr := e.String()
	if n = d.table.resolveAddr(addr); n != nil {
		return
	}

	n, err = nodeFromEndPoint(e)
	if err != nil {
		return nil, err
	}

	err = d.ping(n)
	if err != nil {
		return nil, err
	}

	return n, nil
}

func (d *discovery) checkNode(n *Node) error {
	addr, err := n.udpAddr()
	if err != nil {
		return err
	}

	// add to stage first
	d.mu.Lock()
	d.stage[addr.String()] = n
	d.mu.Unlock()

	// ping
	err = d.ping(n)

	d.mu.Lock()
	delete(d.stage, addr.String())
	d.mu.Unlock()

	if err != nil {
		return err
	}

	// if ping success, then add to table
	d.table.add(n)

	return nil
}

func (d *discovery) getBootNodes(num int) (bootNodes []*Node) {
	for _, btr := range d.booters {
		bootNodes = append(bootNodes, btr.getBootNodes(num)...)
	}

	return
}

func (d *discovery) init() {
	if d.loadBootNodes() {
		d.refresh()
	}
}

func (d *discovery) loadBootNodes() bool {
	var failed int

Load:
	bootNodes := d.getBootNodes(d.BucketSize)

	if len(bootNodes) == 0 {
		failed++
		if failed > 5 {
			// todo
			return false
		}
		goto Load
	}

	for _, n := range bootNodes {
		_ = d.receiveNode(n)
	}

	return true
}

func (d *discovery) findSubTree(distance uint) {
	if d.refreshing {
		return
	}

	if d.loadBootNodes() {
		id := vnode.RandFromDistance(d.node.ID, distance)
		nodes := d.lookup(id, d.BucketSize)
		d.table.addNodes(nodes)
	}
}

func (d *discovery) refresh() {
	d.mu.Lock()
	if d.refreshing {
		d.mu.Unlock()
		return
	}
	d.refreshing = true
	d.mu.Unlock()

	for i := uint(0); i < vnode.IDBits; i++ {
		id := vnode.RandFromDistance(d.node.ID, i)
		nodes := d.lookup(id, d.BucketSize)
		d.table.addNodes(nodes)
	}

	d.mu.Lock()
	d.refreshing = false
	d.mu.Unlock()

	d.cond.Broadcast()
}

func (d *discovery) lookup(target vnode.NodeID, count int) []*Node {
	// is looking
	if !atomic.CompareAndSwapInt32(&d.looking, 0, 1) {
		return nil
	}

	defer atomic.StoreInt32(&d.looking, 0)

	var result = closet{
		pivot: target,
	}

	result.nodes = d.table.findNeighbors(target, int(count))

	if len(result.nodes) == 0 {
		return nil
	}

	asked := make(map[vnode.NodeID]struct{}) // nodes has sent findnode message
	asked[d.node.ID] = struct{}{}
	// all nodes of responsive neighbors, use for filter to ensure the same node pushed once
	seen := make(map[vnode.NodeID]struct{})
	seen[target] = struct{}{}

	const alpha = 3
	reply := make(chan []*Node, alpha)
	queries := 0

Loop:
	for {
		for i := 0; i < len(result.nodes) && queries < alpha; i++ {
			n := result.nodes[i]
			if _, ok := asked[n.ID]; !ok && n.couldFind() {
				asked[n.ID] = struct{}{}
				go d.findNode(target, count, n, reply)
				queries++
			}
		}

		if queries == 0 {
			break
		}

		select {
		case <-d.term:
			break Loop
		case nodes := <-reply:
			queries--
			for _, n := range nodes {
				if n != nil && n.Net == d.node.Net {
					if _, ok := seen[n.ID]; !ok {
						seen[n.ID] = struct{}{}
						result.push(n)
					}
				}
			}
		}
	}

	return result.nodes
}

func (d *discovery) findNode(target vnode.NodeID, count int, n *Node, ch chan<- []*Node) {
	epChan := make(chan []*vnode.EndPoint)
	err := d.socket.findNode(target, count, n, epChan)
	if err != nil {
		ch <- nil
		return
	}

	eps := <-epChan
	curr := make(chan struct{}, 10)
	nch := make(chan *Node)
	var wg sync.WaitGroup
	for _, ep := range eps {
		wg.Add(1)
		go func(ep *vnode.EndPoint) {
			defer wg.Done()
			curr <- struct{}{}
			defer func() {
				<-curr
			}()
			node, e := d.receiveEndPoint(*ep)
			if e != nil {
				return
			}
			nch <- node
		}(ep)
	}

	go func() {
		wg.Wait()
		close(nch)
	}()

	nodes := make([]*Node, 0, len(eps))
	for n = range nch {
		nodes = append(nodes, n)
	}

	ch <- nodes
}
