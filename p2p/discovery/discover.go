package discovery

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p/block"
	"github.com/vitelabs/go-vite/p2p/network"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const seedCount = 20
const seedMaxAge = 7 * 24 * time.Hour

var tExpire = 24 * time.Hour         // nodes have been ping-pong checked during this time period are considered valid
var tRefresh = 1 * time.Hour         // refresh the node table at tRefresh intervals
var storeInterval = 5 * time.Minute  // store nodes in table to db at storeDuration intervals
var checkInterval = 10 * time.Second // check the oldest node in table at checkInterval intervals
var stayInTable = 5 * time.Minute    // minimal duration node stay in table can be store in db
var findInterval = time.Minute
var dbCleanInterval = time.Hour

var errUnsolicitedMsg = errors.New("unsolicited message")

func Priv2NodeID(priv ed25519.PrivateKey) (NodeID, error) {
	pub := priv.PubByte()
	return Bytes2NodeID(pub)
}

// @section Discovery
type Config struct {
	Priv      ed25519.PrivateKey
	DBPath    string
	BootNodes []*Node
	Addr      *net.UDPAddr
	Self      *Node
	NetID     network.ID
}

type Discovery struct {
	cfg         *Config
	bootNodes   []*Node
	self        *Node
	agent       *agent
	tab         *table
	db          *nodeDB
	term        chan struct{}
	refreshing  int32 // atomic, whether indicate node table is refreshing
	refreshDone chan struct{}
	wg          sync.WaitGroup
	blockList   *block.Set
	subs        []chan<- *Node
	log         log15.Logger
}

func New(cfg *Config) (d *Discovery) {
	d = &Discovery{
		cfg:         cfg,
		bootNodes:   cfg.BootNodes,
		self:        cfg.Self,
		tab:         newTable(cfg.Self.ID, cfg.NetID),
		refreshDone: make(chan struct{}),
		blockList:   block.New(1000),
		log:         log15.New("module", "p2p/discv"),
	}

	d.agent = newAgent(&agentConfig{
		Self:    cfg.Self,
		Addr:    cfg.Addr,
		Priv:    cfg.Priv,
		Handler: d.HandleMsg,
	})

	return
}

func (d *Discovery) Start() (err error) {
	d.log.Info(fmt.Sprintf("discovery %s start", d.self))

	db, err := newDB(d.cfg.DBPath, 2, d.self.ID)
	if err != nil {
		d.log.Error(fmt.Sprintf("create p2p db error: %v", err))
	}
	d.db = db

	err = d.agent.start()
	if err != nil {
		return err
	}

	d.term = make(chan struct{})

	d.wg.Add(1)
	common.Go(d.tableLoop)

	return
}

func (d *Discovery) Stop() {
	if d.term == nil {
		return
	}

	select {
	case <-d.term:
	default:
		d.log.Warn(fmt.Sprintf("stop discovery %s", d.self))

		close(d.term)
		d.db.close()
		d.agent.stop()
		d.wg.Wait()

		d.log.Warn(fmt.Sprintf("discovery %s stopped", d.self))
	}
}

func (d *Discovery) Block(ID NodeID, IP net.IP) {
	if !ID.IsZero() {
		d.blockList.Add(ID[:])
	}
	if IP != nil {
		d.blockList.Add(IP)
	}
}

func (d *Discovery) Mark(id NodeID, lifetime int64) {
	// todo
}

func (d *Discovery) Need(n uint) {
	nodes := make([]*Node, n)
	i := d.RandomNodes(nodes)
	d.batchNotify(nodes[:i])
}

func (d *Discovery) tableLoop() {
	defer d.wg.Done()

	checkTicker := time.NewTicker(checkInterval)
	refreshTicker := time.NewTimer(tRefresh)
	storeTicker := time.NewTicker(storeInterval)
	findTicker := time.NewTicker(findInterval)
	dbTicker := time.NewTicker(dbCleanInterval)

	defer checkTicker.Stop()
	defer refreshTicker.Stop()
	defer storeTicker.Stop()
	defer findTicker.Stop()
	defer dbTicker.Stop()

	d.RefreshTable()

	for {
		select {
		case <-refreshTicker.C:
			d.RefreshTable()

		case <-checkTicker.C:
			n := d.tab.pickOldest()
			if n != nil {
				d.agent.ping(n, func(node *Node, err error) {
					if err == nil {
						d.tab.bubble(n)
					} else {
						d.tab.removeNode(n)
					}
				})
			}

		case <-findTicker.C:
			d.lookup(d.self.ID, false)

		case <-storeTicker.C:
			now := time.Now()
			d.tab.traverse(func(n *Node) {
				if now.Sub(n.addAt) > stayInTable {
					d.db.storeNode(n)
				}
			})

		case <-dbTicker.C:
			d.db.cleanStaleNodes()

		case <-d.term:
			return
		}
	}
}

func (d *Discovery) findNode(to *Node, target NodeID, callback func(n *Node, nodes []*Node)) {
	d.agent.findnode(to, target, func(nodes []*Node, err error) {
		callback(to, nodes)
		if len(nodes) == 0 {
			discvLog.Warn(fmt.Sprintf("find %s to %s, got %d neighbors, error: %v", target, to.UDPAddr(), len(nodes), err))
		} else {
			discvLog.Info(fmt.Sprintf("find %s to %s, got %d neighbors", target, to.UDPAddr(), len(nodes)))

			// add as many nodes as possible
			for _, n := range nodes {
				d.tab.addNode(n)
			}
		}
	})
}

func (d *Discovery) RandomNodes(result []*Node) int {
	return d.tab.randomNodes(result)
}

func (d *Discovery) Lookup(id NodeID) []*Node {
	return d.lookup(id, true)
}

func (d *Discovery) lookup(id NodeID, refreshIfNull bool) []*Node {
	var result *neighbors

	for {
		result = d.tab.findNeighbors(id, K)

		if len(result.nodes) > 0 || !refreshIfNull {
			break
		}

		d.RefreshTable()

		// table has refreshed, should not refresh again in a short time
		refreshIfNull = false
	}

	asked := make(map[NodeID]struct{}) // nodes has sent findnode message
	asked[d.self.ID] = struct{}{}

	// all nodes of responsive neighbors, use for filter to ensure the same node pushed once
	hasPushedIntoResult := make(map[NodeID]struct{})

	reply := make(chan []*Node, alpha)
	queries := 0

loop:
	for {
		select {
		case <-d.term:
			break loop
		default:
		}

		for i := 0; i < len(result.nodes) && queries < alpha; i++ {
			n := result.nodes[i]
			if _, ok := asked[n.ID]; !ok {
				asked[n.ID] = struct{}{}
				d.findNode(n, id, func(n *Node, nodes []*Node) {
					reply <- nodes
				})
				queries++
			}
		}

		if queries == 0 {
			break
		}

		select {
		case <-d.term:
			break loop
		case nodes := <-reply:
			queries--
			for _, n := range nodes {
				if n != nil {
					if _, ok := hasPushedIntoResult[n.ID]; !ok {
						hasPushedIntoResult[n.ID] = struct{}{}
						result.push(n)
					}
				}
			}
		}
	}

	return result.nodes
}

// find Node who`s equal id
func (d *Discovery) Resolve(id NodeID) *Node {
	nodes := d.tab.findNeighbors(id, 1).nodes

	if len(nodes) > 0 && id.Equal(nodes[0].ID) {
		return nodes[0]
	}

	nodes = d.Lookup(id)
	for _, n := range nodes {
		if n != nil && id.Equal(n.ID) {
			return n
		}
	}

	return nil
}

func (d *Discovery) HandleMsg(res *packet) {
	if res.msg.isExpired() {
		return
	}

	switch res.code {
	case pingCode:
		monitor.LogEvent("p2p/discv", "ping-receive")
		ping := res.msg.(*Ping)

		node := &Node{
			ID:  res.fromID,
			IP:  res.from.IP,
			UDP: uint16(res.from.Port), // use the remote address
			TCP: ping.TCP,              // extract from the message
		}

		d.db.setLastPing(res.fromID, time.Now())
		d.agent.pong(node, res.hash)
		d.tab.addNode(node)
		d.notify(node)
	case pongCode:
		monitor.LogEvent("p2p/discv", "pong-receive")

		pong := res.msg.(*Pong)
		// get our public IP
		if len(pong.IP) == 0 || bytes.Equal(pong.IP, net.IPv4zero) || bytes.Equal(pong.IP, net.IPv6zero) {
			// do nothing
		} else {
			d.self.IP = pong.IP
		}

		d.db.setLastPong(res.fromID, time.Now())
	case findnodeCode:
		monitor.LogEvent("p2p/discv", "find-receive")

		findMsg := res.msg.(*FindNode)
		nodes := d.tab.findNeighbors(findMsg.Target, K).nodes
		node := &Node{
			ID:  res.fromID,
			IP:  res.from.IP,
			UDP: uint16(res.from.Port), // use the remote address
		}

		d.agent.sendNeighbors(node, nodes)
		d.tab.addNode(node)
	case neighborsCode:
		monitor.LogEvent("p2p/discv", "neighbors-receive")

	default:
		d.agent.send(&sendPkt{
			addr: res.from,
			code: exceptionCode,
			msg: &Exception{
				Code: eUnKnown,
			},
		})
	}
}

func (d *Discovery) RefreshTable() {
	if !atomic.CompareAndSwapInt32(&d.refreshing, 0, 1) {
		select {
		case <-d.term:
		case <-d.refreshDone:
		}
		return
	}

	discvLog.Info("refresh table")

	monitor.LogEvent("p2p/discv", "refreshTable")

	// do refresh routine
	d.tab.initRand()
	d.loadInitNodes()

	nodes := d.lookup(d.self.ID, false)
	d.batchNotify(nodes)

	// find random NodeID in order to improve the defense of eclipse attack
	for i := 0; i < alpha; i++ {
		var id NodeID
		rand.Read(id[:])
		nodes = d.lookup(id, false)
		d.batchNotify(nodes)
	}

	// set the right state after refresh
	atomic.CompareAndSwapInt32(&d.refreshing, 1, 0)
	close(d.refreshDone)
	d.refreshDone = make(chan struct{})
	discvLog.Info("refresh table done")
}

func (d *Discovery) loadInitNodes() {
	nodes := d.db.randomNodes(seedCount, seedMaxAge) // get random nodes from db
	nodes = append(nodes, d.bootNodes...)

	discvLog.Info(fmt.Sprintf("got %d nodes from db", len(nodes)))

	for _, node := range nodes {
		d.tab.addNode(node)
	}
}

func (d *Discovery) Nodes() []*Node {
	return d.tab.nodes()
}

func (d *Discovery) SubNodes(ch chan<- *Node) {
	d.subs = append(d.subs, ch)
}

func (d *Discovery) UnSubNodes(ch chan<- *Node) {
	for i, c := range d.subs {
		if c == ch {
			copy(d.subs[i:], d.subs[i+1:])
			d.subs = d.subs[:len(d.subs)-1]
		}
	}
}

func (d *Discovery) notify(node *Node) {
	for _, ch := range d.subs {
		select {
		case ch <- node:
		default:
		}
	}
}

func (d *Discovery) batchNotify(nodes []*Node) {
	for _, node := range nodes {
		d.notify(node)
	}
}
