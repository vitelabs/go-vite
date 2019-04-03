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
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p/config"
	"github.com/vitelabs/go-vite/p2p/vnode"
)

const seedCount = 50
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
var errPingNotFailed = errors.New("failed to ping node")

// Discovery is the interface to discovery other node
type Discovery interface {
	Start() error
	Stop() error
	Delete(id vnode.NodeID)
	GetNodes(count int) []vnode.Node
	Resolve(id vnode.NodeID)
	Verify(node vnode.Node)
	SetFinder(f Finder)
}

type db interface {
	Store(node *Node) (err error)
	ReadNodes(count int, maxAge time.Duration) []*Node
	Close() error
	Clean(expiration time.Duration)
	Remove(ID vnode.NodeID)
}

type discovery struct {
	config.Config

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

func (d *discovery) GetNodes(count int) []vnode.Node {
	// can NOT use rw.RLock(), will panic
	d.mu.Lock()
	for d.refreshing {
		d.cond.Wait()
	}
	d.mu.Unlock()

	return d.finder.GetNodes(count)
}

func (d *discovery) SetFinder(f Finder) {
	d.finder.UnSub(d.table)

	d.finder = f
	f.Sub(d.table)
}

func (d *discovery) Delete(id vnode.NodeID) {
	d.table.remove(id)
	d.db.Remove(id)
}

func (d *discovery) Resolve(id vnode.NodeID) {

}

func (d *discovery) Verify(node vnode.Node) {
	n := d.table.resolve(node.ID)
	if n != nil {
		_ = d.ping(n)
	}
}

// New create a Discovery implementation
func New(cfg config.Config) Discovery {
	d := &discovery{
		Config: cfg,
		stage:  make(map[string]*Node),
		log:    log15.New("module", "p2p/discv"),
	}

	d.cond = sync.NewCond(&d.mu)

	d.socket = newAgent(cfg.PrivateKey, &cfg.Node, d.handle)

	d.table = newTable(&cfg.Node, bucketSize, bucketNum, newListBucket, d)

	return d
}

func (d *discovery) Start() (err error) {
	if !atomic.CompareAndSwapInt32(&d.running, 0, 1) {
		return errDiscoveryIsRunning
	}

	ndb, err := newDB(d.Config.DataDir, 3, d.Node.ID)
	if err != nil {
		return
	}
	d.db = ndb

	d.booters = append(d.booters, newDBBooter(d.db))
	if len(d.BootSeed) > 0 {
		d.booters = append(d.booters, newNetBooter(&d.Node, d.BootSeed))
	}
	if len(d.BootNodes) > 0 {
		var bt booter
		bt, err = newCfgBooter(d.BootNodes)
		if err != nil {
			return
		}
		d.booters = append(d.booters, bt)
	}

	err = d.socket.start()
	if err != nil {
		return
	}

	d.term = make(chan struct{})

	d.wg.Add(1)
	go d.tableLoop()

	return
}

func (d *discovery) Stop() (err error) {
	if atomic.CompareAndSwapInt32(&d.running, 1, 0) {
		close(d.term)
		err = d.socket.stop()
		err = d.db.Close()
		d.wg.Wait()

		return
	}

	return errDiscoveryIsStopped
}

// ping and update n, block
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
		return errPingNotFailed
	}

	if n2.Net != d.Node.Net {
		return errDifferentNet
	}

	n.update(n2)
	return nil
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
				go d.ping(node)
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

func (d *discovery) handle(res *packet) {
	switch res.c {
	case codePing:
		n := genNodeFromPing(res)
		n2 := d.table.resolveAddr(n.Address())

		if n2 != nil {
			if n.Net != d.Node.Net {
				d.table.remove(res.id)
			} else {
				n2.update(n)
				d.table.bubble(n2.ID)
				_ = d.socket.pong(res.hash.Bytes(), n)
			}
		} else {
			if n.Net != d.Node.Net {
				return
			}
			// add to table
			d.table.add(n)
			_ = d.socket.pong(res.hash.Bytes(), n)
		}

	case codePong:
		// nothing
	case codeFindnode:
		find := res.body.(*findnode)
		nodes := d.table.findNeighbors(find.target, int(find.count))
		eps := make([]vnode.EndPoint, len(nodes))
		for i, n := range nodes {
			eps[i] = n.EndPoint
		}
		_ = d.socket.sendNodes(eps, res.from)

	case codeNeighbors:
		d.table.bubble(res.id)
	}
}

// seeNewNode will check a new node
func (d *discovery) seeNewNode(n *Node) error {
	if n.Net != d.Node.Net {
		return errDifferentNet
	}

	if n.shouldCheck() {
		return d.checkNode(n)
	} else {
		d.table.add(n)
	}

	return nil
}

func (d *discovery) checkNode(n *Node) error {
	addr, err := n.udpAddr()
	if err != nil {
		return err
	}

	d.mu.Lock()
	d.stage[addr.String()] = n
	d.mu.Unlock()

	err = d.ping(n)

	d.mu.Lock()
	delete(d.stage, addr.String())
	d.mu.Unlock()

	if err != nil {
		return err
	}

	d.table.add(n)

	return nil
}

func (d *discovery) init() {
	var nodes []*Node
	var failed int
Load:
	for _, btr := range d.booters {
		nodes = append(nodes, btr.getBootNodes(seedCount)...)
	}

	if len(nodes) == 0 {
		failed++
		if failed > 5 {
			// todo
		}
		goto Load
	}

	for _, n := range nodes {
		_ = d.seeNewNode(n)
	}

	d.refresh()
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
		id := vnode.RandFromDistance(d.Node.ID, i)
		nodes := d.lookup(id, bucketSize)
		d.table.addNodes(nodes)
	}

	d.cond.Signal()
}

func (d *discovery) lookup(target vnode.NodeID, count uint32) []*Node {
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
	asked[d.Node.ID] = struct{}{}
	// all nodes of responsive neighbors, use for filter to ensure the same node pushed once
	seen := make(map[vnode.NodeID]struct{})
	seen[target] = struct{}{}

	const alpha = 10
	reply := make(chan []*Node, alpha)
	queries := 0

Loop:
	for {
		for i := 0; i < len(result.nodes) && queries < alpha; i++ {
			n := result.nodes[i]
			if _, ok := asked[n.ID]; !ok && n.couldFind() {
				asked[n.ID] = struct{}{}
				d.findNode(target, count, n, reply)
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
				if n != nil && n.Net == d.Node.Net {
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

func (d *discovery) findNode(target vnode.NodeID, count uint32, n *Node, ch chan<- []*Node) {
	epChan := make(chan []vnode.EndPoint)
	err := d.socket.findNode(target, count, n, epChan)
	if err != nil {
		ch <- nil
		return
	}

	eps := <-epChan
	nch := make(chan *Node)
	var wg sync.WaitGroup
	for _, ep := range eps {
		if n = d.table.resolveAddr(ep.String()); n != nil {
			nch <- n
		} else {
			wg.Add(1)
			go func(ep vnode.EndPoint) {
				defer wg.Done()

				udp, e := net.ResolveUDPAddr("udp", ep.String())
				if e != nil {
					return
				}

				var node = &Node{
					Node: vnode.Node{
						EndPoint: ep,
					},
					addr: udp,
				}

				_ = d.socket.ping(node, nch)
			}(ep)
		}
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
