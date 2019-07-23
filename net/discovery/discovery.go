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

	"github.com/vitelabs/go-vite/crypto/ed25519"

	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/net/vnode"
)

const seedMaxAge = 7 * 24 * 3600       // 7d
const tRefresh = 24 * time.Hour        // refresh the node table at tRefresh intervals
const storeInterval = 5 * time.Minute  // store nodes in table to db at storeDuration intervals
const checkInterval = 10 * time.Second // check the oldest node in table at checkInterval intervals
const checkExpiration = 3600           // should check again if last check is an hour ago
const stayInTable = 300                // minimal duration node stay in table can be store in db
const dbCleanInterval = time.Hour

var errDiscoveryIsRunning = errors.New("discovery is running")
var errDiscoveryIsStopped = errors.New("discovery is stopped")
var errDifferentNet = errors.New("different net")
var errPingFailed = errors.New("failed to ping node")

var discvLog = log15.New("module", "discovery")

type NodeDB interface {
	StoreNode(node *vnode.Node) (err error)
	RemoveNode(id vnode.NodeID)
	ReadNodes(expiration int64) []*vnode.Node
	RetrieveActiveAt(id vnode.NodeID) int64
	StoreActiveAt(id vnode.NodeID, v int64)
	RetrieveCheckAt(id vnode.NodeID) int64
	StoreCheckAt(id vnode.NodeID, v int64)
	Close() error
	Clean(expiration int64)
}

type discvDB struct {
	NodeDB
}

func (db *discvDB) StoreNode(node *Node) (err error) {
	err = db.NodeDB.StoreNode(&node.Node)
	db.StoreActiveAt(node.ID, node.activeAt)
	db.StoreCheckAt(node.ID, node.checkAt)

	return
}

func (db *discvDB) ReadNodes(expiration int64) (nodes []*Node) {
	vnodes := db.NodeDB.ReadNodes(expiration)
	nodes = make([]*Node, len(vnodes))
	for i, n := range vnodes {
		nodes[i] = &Node{
			Node:     *n,
			checkAt:  db.RetrieveCheckAt(n.ID),
			activeAt: db.RetrieveActiveAt(n.ID),
		}
	}

	return
}

type Discovery struct {
	node *vnode.Node

	bootNodes []string
	bootSeeds []string

	booters []booter

	table nodeTable

	finder Finder

	// nodes wait to check
	stage map[string]*Node // key is the communication address, not endpoint
	mu    sync.Mutex

	socket socket

	db *discvDB

	running int32
	term    chan struct{}

	looking int32 // is looking self

	refreshing bool
	cond       *sync.Cond

	wg sync.WaitGroup

	log log15.Logger
}

func (d *Discovery) Nodes() []*vnode.Node {
	nodes := d.table.nodes(0)
	vnodes := make([]*vnode.Node, len(nodes))
	for i, n := range nodes {
		vnodes[i] = &n.Node
	}

	return vnodes
}

func (d *Discovery) SubscribeNode(receiver func(n *vnode.Node)) (subId int) {
	return d.table.Sub(receiver)
}
func (d *Discovery) Unsubscribe(subId int) {
	d.table.UnSub(subId)
}

func (d *Discovery) GetNodes(count int) (nodes []*vnode.Node) {
	for _, n := range d.table.nodes(count) {
		nodes = append(nodes, &n.Node)
	}

	return
}

func (d *Discovery) SetFinder(f Finder) {
	if d.finder != nil {
		d.finder.UnSub(d.table)
	}

	f.SetResolver(d)
	d.finder = f
	d.finder.Sub(d.table)
}

func (d *Discovery) Delete(id vnode.NodeID, reason error) {
	d.table.remove(id)

	if d.db != nil {
		d.db.RemoveNode(id)
	}

	d.log.Info(fmt.Sprintf("remove node %s from table: %v", id.String(), reason))
}

func (d *Discovery) Resolve(id vnode.NodeID) (ret *vnode.Node) {
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
func New(peerKey ed25519.PrivateKey, node *vnode.Node, bootNodes, bootSeeds []string, listenAddress string, db NodeDB) *Discovery {
	d := &Discovery{
		node:       node,
		bootNodes:  bootNodes,
		bootSeeds:  bootSeeds,
		booters:    nil,
		table:      nil,
		finder:     nil,
		stage:      make(map[string]*Node),
		mu:         sync.Mutex{},
		socket:     nil,
		db:         nil,
		running:    0,
		term:       nil,
		looking:    0,
		refreshing: false,
		cond:       nil,
		wg:         sync.WaitGroup{},
		log:        discvLog,
	}
	if db != nil {
		d.db = &discvDB{
			db,
		}
	}

	d.cond = sync.NewCond(&d.mu)

	d.socket = newAgent(peerKey, d.node, listenAddress, d.handle)

	d.table = newTable(d.node.ID, node.Net, newListBucket, d)

	return d
}

func (d *Discovery) Start() (err error) {
	if !atomic.CompareAndSwapInt32(&d.running, 0, 1) {
		return errDiscoveryIsRunning
	}

	// retrieve boot nodes
	if d.db != nil {
		d.booters = append(d.booters, newDBBooter(d.db))
	}
	if len(d.bootSeeds) > 0 {
		d.booters = append(d.booters, newNetBooter(d.node, d.bootSeeds))
	}
	if len(d.bootNodes) > 0 {
		var bt booter
		bt, err = newCfgBooter(d.bootNodes, d.node)
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

func (d *Discovery) Stop() (err error) {
	if atomic.CompareAndSwapInt32(&d.running, 1, 0) {
		close(d.term)
		d.wg.Wait()

		err = d.socket.stop()

		if d.db != nil {
			err = d.db.Close()
		}

		return
	}

	return errDiscoveryIsStopped
}

// ping and update a node, will be blocked, so should invoked by goroutine
func (d *Discovery) ping(n *Node) error {
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

	n.checkAt = time.Now().Unix()
	n.update(n2)
	return nil
}

func (d *Discovery) pingDelete(n *Node) {
	if d.ping(n) != nil {
		d.table.resolve(n.ID)
	}
}

func (d *Discovery) tableLoop() {
	defer d.wg.Done()

	d.init()

	checkTicker := time.NewTicker(checkInterval)
	refreshTicker := time.NewTimer(tRefresh)
	defer checkTicker.Stop()
	defer refreshTicker.Stop()

	var storeChan, cleanChan <-chan time.Time
	if d.db != nil {
		storeTicker := time.NewTicker(storeInterval)
		cleanTicker := time.NewTicker(dbCleanInterval)
		defer storeTicker.Stop()
		defer cleanTicker.Stop()

		storeChan = storeTicker.C
		cleanChan = cleanTicker.C
	}

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

		case <-storeChan:
			d.table.store(d.db)

		case <-cleanChan:
			d.db.Clean(seedMaxAge)
		}
	}

	if d.db != nil {
		d.table.store(d.db)
	}
}

func (d *Discovery) findLoop() {
	defer d.wg.Done()

	duration := 10 * time.Second
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

Loop:
	for {
		select {
		case <-d.term:
			break Loop

		case <-ticker.C:
			if d.table.size() == 0 {
				d.init()
			} else if distance := d.table.subTreeToFind(); distance > 0 {
				d.findSubTree(distance)
			}
		}
	}
}

func (d *Discovery) handle(pkt *packet) {
	defer recyclePacket(pkt)

	d.table.bubble(pkt.id)

	switch pkt.c {
	case codePing:
		n := nodeFromPing(pkt)
		if n.Net == d.node.Net {
			_ = d.socket.pong(pkt.hash, n)
		}

		d.table.add(n)

	case codePong:
		// nothing
		// pong will be handle by requestPool

	case codeFindnode:
		find := pkt.body.(*findnode)

		var eps []*vnode.EndPoint
		if d.finder != nil {
			eps = d.finder.FindNeighbors(pkt.id, find.target, find.count/2)
		}
		if len(eps) < find.count {
			nodes := d.table.findNeighbors(find.target, find.count-len(eps))
			for _, n := range nodes {
				eps = append(eps, &n.EndPoint)
			}
		}

		_ = d.socket.sendNodes(eps, pkt.from)

	case codeNeighbors:
		// nothing
	}
}

// receiveNode get from bootNodes or ping message, check and put into table
func (d *Discovery) receiveNode(n *Node) error {
	if n.Net != d.node.Net {
		return errDifferentNet
	}

	if d.table.resolve(n.ID) != nil {
		return nil
	}

	// 30min
	if time.Now().Unix()-n.checkAt > 30*60 {
		return d.checkNode(n)
	}

	d.table.add(n)
	return nil
}

// receiveEndPoint from neighbors message
func (d *Discovery) receiveEndPoint(e vnode.EndPoint) (n *Node, err error) {
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

func (d *Discovery) checkNode(n *Node) error {
	addr, err := n.udpAddr()
	if err != nil {
		return err
	}

	d.mu.Lock()
	if _, ok := d.stage[addr.String()]; ok {
		d.mu.Unlock()
		return nil
	}
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

func (d *Discovery) getBootNodes(num int) (bootNodes []*Node) {
	for _, btr := range d.booters {
		bootNodes = append(bootNodes, btr.getBootNodes(num)...)
	}

	return
}

func (d *Discovery) init() {
	if d.loadBootNodes() {
		d.refresh()
	}
}

func (d *Discovery) loadBootNodes() bool {
	var failed int

Load:
	bootNodes := d.getBootNodes(bucketSize)

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

func (d *Discovery) findSubTree(distance uint) {
	if d.refreshing {
		return
	}

	id := vnode.RandFromDistance(d.node.ID, distance)
	nodes := d.lookup(id, bucketSize)
	d.table.addNodes(nodes)
}

func (d *Discovery) refresh() {
	d.mu.Lock()
	if d.refreshing {
		d.mu.Unlock()
		return
	}
	d.refreshing = true
	d.mu.Unlock()

	for i := uint(0); i < vnode.IDBits; i++ {
		id := vnode.RandFromDistance(d.node.ID, i)
		nodes := d.lookup(id, bucketSize)
		d.table.addNodes(nodes)
		// have no enough nodes
		if len(nodes) < bucketSize {
			break
		}
	}

	d.mu.Lock()
	d.refreshing = false
	d.mu.Unlock()

	d.cond.Broadcast()
}

func (d *Discovery) lookup(target vnode.NodeID, count int) (result []*Node) {
	// is looking
	if !atomic.CompareAndSwapInt32(&d.looking, 0, 1) {
		return nil
	}

	defer atomic.StoreInt32(&d.looking, 0)

	nodes := d.table.findNeighbors(target, count)

	if len(nodes) == 0 {
		return
	}

	asked := make(map[vnode.NodeID]struct{}) // nodes has sent findnode message
	asked[d.node.ID] = struct{}{}
	asked[target] = struct{}{}

	// all nodes of responsive neighbors, use for filter to ensure the same node pushed once
	seen := make(map[vnode.NodeID]struct{})
	seen[d.node.ID] = struct{}{}
	for _, n := range nodes {
		seen[n.ID] = struct{}{}
	}

	const alpha = 5
	reply := make(chan []*Node, alpha)
	queries := 0

Loop:
	for {
		for i := 0; i < len(nodes) && queries < alpha; i++ {
			n := nodes[i]
			if _, ok := asked[n.ID]; !ok {
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
		case ns := <-reply:
			queries--
			for _, n := range ns {
				if n != nil && n.Net == d.node.Net {
					if _, ok := seen[n.ID]; !ok {
						seen[n.ID] = struct{}{}
						result = append(result, n)
					}
				}
			}
		}
	}

	return
}

func (d *Discovery) findNode(target vnode.NodeID, count int, n *Node, ch chan<- []*Node) {
	epChan := make(chan []*vnode.EndPoint)
	err := d.socket.findNode(target, count, n, epChan)
	if err != nil {
		ch <- nil
		return
	}

	eps := <-epChan
	var nodes []*Node
	if len(eps) > 0 {
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

		nodes = make([]*Node, 0, len(eps))
		for n = range nch {
			nodes = append(nodes, n)
		}
	}

	ch <- nodes
}
