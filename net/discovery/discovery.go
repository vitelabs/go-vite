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
	"math/rand"
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

type pingStatus = byte

const (
	ping_ing     pingStatus = 0
	ping_success pingStatus = 1
	ping_failed  pingStatus = 2
)

type checkEndPointResult struct {
	time      int64
	status    pingStatus
	node      *Node
	err       error
	callbacks []func(err error)
}

type Discovery struct {
	node *vnode.Node

	bootNodes []string
	bootSeeds []string

	booters []booter

	table nodeTable

	finder Finder

	stage map[string]*checkEndPointResult
	mu    sync.Mutex

	socket socket

	db *discvDB

	running int32
	term    chan struct{}

	looking int32 // is looking self

	refreshing bool

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

func (d *Discovery) NodesCount() int {
	return d.table.size()
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

// New create a Discovery implementation
func New(peerKey ed25519.PrivateKey, node *vnode.Node, bootNodes, bootSeeds []string, listenAddress string, db NodeDB) *Discovery {
	d := &Discovery{
		node:       node,
		bootNodes:  bootNodes,
		bootSeeds:  bootSeeds,
		booters:    nil,
		table:      nil,
		finder:     nil,
		stage:      make(map[string]*checkEndPointResult),
		socket:     nil,
		db:         nil,
		running:    0,
		term:       nil,
		looking:    0,
		refreshing: false,
		wg:         sync.WaitGroup{},
		log:        discvLog,
	}
	if db != nil {
		d.db = &discvDB{
			db,
		}
	}

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
func (d *Discovery) ping(n *Node, callback func(err error)) {
	address := n.Address()
	now := time.Now().Unix()

	d.mu.Lock()
	if r, ok := d.stage[address]; ok {
		if r.status == ping_ing {
			r.callbacks = append(r.callbacks, callback)
			d.mu.Unlock()
			return
		} else if r.status == ping_failed && now-r.time < 10*60 {
			err := r.err
			d.mu.Unlock()
			callback(err)
			return
		} else if r.status == ping_success && now-r.time < 10*60 {
			n.update(r.node)
			d.mu.Unlock()
			callback(nil)
			return
		}
	}

	d.stage[address] = &checkEndPointResult{
		time:   now,
		status: ping_ing,
		node:   n,
		err:    nil,
		callbacks: []func(err error){
			callback,
		},
	}
	d.mu.Unlock()

	d.socket.ping(n, func(node *Node, err error) {
		d.mu.Lock()

		r, ok := d.stage[address]
		if !ok {
			r = &checkEndPointResult{}
			d.stage[address] = r
		}

		r.time = time.Now().Unix()
		if err != nil {
			r.status = ping_failed
			r.err = err
			r.node = n
		} else {
			n.update(node)
			r.status = ping_success
			r.node = node
		}

		callbacks := r.callbacks
		d.mu.Unlock()

		for _, cb := range callbacks {
			cb(r.err)
		}

		//if r.err != nil {
		//	discvLog.Warn(fmt.Sprintf("failed to check %s: %v\n", n, r.err))
		//}
	})
}

func (d *Discovery) pingDelete(n *Node) {
	d.ping(n, func(err error) {
		if err != nil {
			d.table.remove(n.ID)
		}
	})
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

	duration := 30 * time.Second
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
	exist := d.table.bubble(pkt.id)

	switch pkt.c {
	case codePing:
		//discvLog.Info(fmt.Sprintf("receive ping from %s", pkt.from.String()))
		n := nodeFromPing(pkt)
		if n.Net == d.node.Net {
			_ = d.socket.pong(pkt.hash, n)
		}

		if !exist {
			d.table.add(n)
		}

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

		//discvLog.Info(fmt.Sprintf("send %d ep to %s", len(eps), pkt.from.String()))
		_ = d.socket.sendNodes(eps, pkt.from)

	case codeNeighbors:
		// nothing
	}
}

// receiveNode get from bootNodes or ping message, check and put into table
func (d *Discovery) receiveNode(n *Node) {
	if n.Net != d.node.Net {
		return
	}

	if d.table.resolve(n.ID) != nil {
		return
	}

	addr, err := n.udpAddr()
	if err != nil {
		return
	}

	if d.table.resolveAddr(addr.String()) != nil {
		return
	}

	if n.needCheck() {
		d.ping(n, func(err error) {
			if err != nil {
				return
			}

			d.table.add(n)
		})
	} else {
		d.table.add(n)
	}
}

// receiveEndPoint from neighbors message
func (d *Discovery) receiveEndPoint(e *vnode.EndPoint) (valid bool) {
	addr := e.String()
	if d.table.resolveAddr(addr) != nil {
		return false
	}

	node, err := nodeFromEndPoint(e)
	if err != nil {
		return false
	}

	d.ping(node, func(err error) {
		if err != nil {
			return
		}

		d.table.add(node)
	})

	return true
}

//func (d *Discovery) checkNode(n *Node) error {
//	_, err := n.udpAddr()
//	if err != nil {
//		return err
//	}
//
//	err = d.ping(n)
//
//	if err != nil {
//		return err
//	}
//
//	// if ping success, then add to table
//	d.table.add(n)
//
//	return nil
//}

func (d *Discovery) getBootNodes(num int) (bootNodes []*Node) {
	for _, btr := range d.booters {
		bootNodes = append(bootNodes, btr.getBootNodes(num)...)
	}

	return
}

func (d *Discovery) init() {
	discvLog.Info("init")
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

	discvLog.Info(fmt.Sprintf("load %d bootNodes", len(bootNodes)))

	for _, n := range bootNodes {
		d.receiveNode(n)
	}

	return true
}

func (d *Discovery) findSubTree(distance uint) {
	if d.refreshing {
		return
	}

	id := vnode.RandFromDistance(d.node.ID, distance)
	d.lookup(id, bucketSize)
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
		d.lookup(id, bucketSize)
	}

	d.mu.Lock()
	d.refreshing = false
	d.mu.Unlock()
}

func (d *Discovery) lookup(target vnode.NodeID, count int) {
	// is looking
	if !atomic.CompareAndSwapInt32(&d.looking, 0, 1) {
		return
	}

	defer atomic.StoreInt32(&d.looking, 0)

	nodes := d.table.findSource(target, count)
	if len(nodes) == 0 {
		return
	}

	//discvLog.Info(fmt.Sprintf("retrieve %d nodes to find %s", len(nodes), target))

	curr := make(chan struct{}, 10)

	for i := 0; i < len(nodes); i++ {
		ratio := d.findNode(target, count, nodes[i], curr)
		if ratio < 0.3 {
			if rand.Intn(10) < 5 {
				break
			}
		}
	}
}

func (d *Discovery) findNode(target vnode.NodeID, count int, n *Node, curr chan struct{}) (ratio float64) {
	epChan, err := d.socket.findNode(target, count, n)
	if err != nil {
		return
	}

	total := 0
	var valid int32 = 0
	for eps := range epChan {
		total += len(eps)
		if len(eps) > 0 {
			for _, ep := range eps {
				curr <- struct{}{}
				go d.receiveEndPointCurr(ep, curr, &valid)
			}
		}
	}

	//discvLog.Info(fmt.Sprintf("receive %d eps from %s", total, n))

	n.findDone()

	if total == 0 {
		return 0
	}

	return float64(valid) / float64(total)
}

func (d *Discovery) receiveEndPointCurr(ep *vnode.EndPoint, ch <-chan struct{}, validCount *int32) {
	valid := d.receiveEndPoint(ep)
	if valid {
		atomic.AddInt32(validCount, 1)
	}

	<-ch
}
