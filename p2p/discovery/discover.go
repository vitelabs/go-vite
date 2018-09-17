package discovery

import (
	"crypto/rand"
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p/block"
	"github.com/vitelabs/go-vite/p2p/nat"
	"net"
	"sync"
	"time"
)

const seedCount = 20
const seedMaxAge = 7 * 24 * time.Hour

var tExpire = 24 * time.Hour          // nodes have been ping-pong checked during this time period are considered valid
var tRefresh = 1 * time.Hour          // refresh the node table at tRefresh intervals
var storeInterval = 5 * time.Minute   // store nodes in table to db at storeDuration intervals
var checkInterval = 3 * time.Minute   // check the oldest node in table at checkInterval intervals
var minStayDuration = 5 * time.Minute // minimal duration node stay in table can be store in db

func Priv2NodeID(priv ed25519.PrivateKey) (NodeID, error) {
	pub := priv.PubByte()
	return Bytes2NodeID(pub)
}

type discvAgent interface {
	start()
	stop()
	ping(node *Node) error
	pong(node *Node, ack types.Hash) error
	findnode(n *Node, ID NodeID) (nodes []*Node, err error)
	sendNeighbors(n *Node, nodes []*Node) error
	need(p *packet) bool
	wait(NodeID, packetCode, waitIsDone) error
}

// @section config
type Config struct {
	Priv      ed25519.PrivateKey
	DBPath    string
	BootNodes []*Node
	Conn      *net.UDPConn
}

type Discovery struct {
	bootNodes   []*Node
	self        *Node
	selfLock    sync.RWMutex
	agent       discvAgent
	pool        wtList
	tab         *table
	db          *nodeDB
	stop        chan struct{}
	lock        sync.RWMutex
	refreshing  bool // use for refresh node table
	refreshDone chan struct{}
	wg          sync.WaitGroup
	blockList   *block.CuckooSet
}

func (d *Discovery) Start() {
	discvLog.Info(fmt.Sprintf("discovery server start %s\n", d.self))

	d.wg.Add(1) // wait for NAT mapping done
	go d.mapping()

	go d.keepTable()

	go d.agent.start()

	<-d.stop
	// clean
	d.db.close()
	d.wg.Wait()
}

func (d *Discovery) Self() *Node {
	d.selfLock.RLock()
	defer d.selfLock.RUnlock()
	return d.self
}

func (d *Discovery) keepTable() {
	checkTicker := time.NewTicker(checkInterval)
	refreshTicker := time.NewTimer(tRefresh)
	storeTicker := time.NewTicker(storeInterval)

	defer checkTicker.Stop()
	defer refreshTicker.Stop()
	defer storeTicker.Stop()

	go d.RefreshTable()

loop:
	for {
		select {
		case <-refreshTicker.C:
			go d.RefreshTable()

		case <-checkTicker.C:
			n := d.tab.pickOldest()
			if n != nil {
				go func() {
					err := d.agent.ping(n)
					if err != nil {
						d.tab.removeNode(n)
					} else {
						d.tab.bubble(n)
					}
				}()
			}

		case <-storeTicker.C:
			now := time.Now()
			go d.tab.traverse(func(n *Node) {
				if now.Sub(n.addAt) > minStayDuration {
					d.db.storeNode(n)
				}
			})

		case <-d.stop:
			break loop
		}
	}
}

func (d *Discovery) findNode(to *Node, target NodeID) (nodes []*Node, err error) {
	// if the last ping-pong checked has too long, then do ping-pong check again
	if !d.db.hasChecked(to.ID) {
		discvLog.Info(fmt.Sprintf("find %s to %s should ping/pong check first\n", target, to.UDPAddr()))

		err = d.agent.ping(to)
		if err != nil {
			discvLog.Error(fmt.Sprintf("ping %s before find %s \n", to.UDPAddr(), target), "error", err)
			return
		}

		d.agent.wait(to.ID, pingCode, func(Message) bool {
			return true
		})
	}

	nodes, err = d.agent.findnode(to, target)
	findFails := d.db.getFindNodeFails(to.ID)
	if err != nil || len(nodes) == 0 {
		findFails++
		d.db.setFindNodeFails(to.ID, findFails)
		discvLog.Info(fmt.Sprintf("find %s to %s fails\n", target, to.UDPAddr()), "error", err, "neighbors", len(nodes))

		if findFails > maxFindFails {
			d.tab.removeNode(to)
		}
	} else {
		discvLog.Info(fmt.Sprintf("find %s to %s success\n", target, to.UDPAddr()), "error", err, "neighbors", len(nodes))

		// add as many nodes as possible
		for _, n := range nodes {
			d.tab.addNode(n)
		}

		if findFails > 0 {
			findFails--
			d.db.setFindNodeFails(to.ID, findFails)
		}
	}

	return nodes, err
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
	asked[d.Self().ID] = struct{}{}

	// all nodes of responsive neighbors, use for filter to ensure the same node pushed once
	hasPushedIntoResult := make(map[NodeID]struct{})

	reply := make(chan []*Node, alpha)
	queries := 0

	for {
		for i := 0; i < len(result.nodes) && queries < alpha; i++ {
			n := result.nodes[i]
			if _, ok := asked[n.ID]; !ok {
				asked[n.ID] = struct{}{}
				go func() {
					nodes, _ := d.findNode(n, id)
					reply <- nodes
				}()
				queries++
			}
		}

		if queries == 0 {
			break
		}

		for _, n := range <-reply {
			if n != nil {
				if _, ok := hasPushedIntoResult[n.ID]; !ok {
					hasPushedIntoResult[n.ID] = struct{}{}
					result.push(n)
				}
			}
		}

		queries--
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

var errUnsolicitedMsg = errors.New("unsolicited message")
var errMsgExpired = errors.New("message has expired")
var errUnkownMsg = errors.New("unknown message")
var errExtractPing = errors.New("extract message error, should be ping")
var errExtractFindNode = errors.New("extract message error, should be findnode")

func (d *Discovery) HandleMsg(res *packet) error {
	if res.msg.isExpired() {
		return errMsgExpired
	}

	discvLog.Info(fmt.Sprintf("receive %s from %s@%s\n", res.code, res.fromID, res.from))

	switch res.code {
	case pingCode:
		monitor.LogEvent("p2p/discv", "ping-receive")
		ping, ok := res.msg.(*Ping)
		if !ok {
			return errExtractPing
		}

		node := &Node{
			ID:  res.fromID,
			IP:  res.from.IP,
			UDP: uint16(res.from.Port), // use the remote address
			TCP: ping.TCP,              // extract from the message
		}

		d.db.setLastPing(res.fromID, time.Now())
		d.agent.pong(node, res.hash)
		d.agent.need(res)

		if !d.db.hasChecked(res.fromID) {
			d.agent.ping(node)
		}
		d.tab.addNode(node)
	case pongCode:
		monitor.LogEvent("p2p/discv", "pong-receive")

		if !d.agent.need(res) {
			return errUnsolicitedMsg
		}
		d.db.setLastPong(res.fromID, time.Now())
	case findnodeCode:
		monitor.LogEvent("p2p/discv", "find-receive")

		if !d.db.hasChecked(res.fromID) {
			return errUnsolicitedMsg
		}

		findMsg, ok := res.msg.(*FindNode)
		if !ok {
			return errExtractFindNode
		}

		nodes := d.tab.findNeighbors(findMsg.Target, K).nodes
		node := &Node{
			ID:  res.fromID,
			IP:  res.from.IP,
			UDP: uint16(res.from.Port), // use the remote address
		}

		d.agent.sendNeighbors(node, nodes)
	case neighborsCode:
		monitor.LogEvent("p2p/discv", "neighbors-receive")

		if !d.agent.need(res) {
			return errUnsolicitedMsg
		}
	default:
		return errUnkownMsg
	}

	return nil
}

func (d *Discovery) RefreshTable() {
	d.lock.Lock()
	if d.refreshing {
		d.lock.Unlock()
		<-d.refreshDone
		return
	}
	d.refreshing = true
	d.lock.Unlock()

	// do refresh routine
	d.tab.initRand()
	d.loadInitNodes()
	d.lookup(d.self.ID, false)

	// find random NodeID in order to improve the defense of eclipse attack
	for i := 0; i < alpha; i++ {
		var id NodeID
		rand.Read(id[:])
		d.lookup(id, false)
	}

	// set the right state after refresh
	close(d.refreshDone)
	d.refreshing = false
	d.refreshDone = make(chan struct{})
}

func (d *Discovery) loadInitNodes() {
	nodes := d.db.randomNodes(seedCount, seedMaxAge) // get random nodes from db
	nodes = append(nodes, d.bootNodes...)
	for _, node := range nodes {
		d.tab.addNode(node)
	}
}

func (d *Discovery) ID() NodeID {
	return d.Self().ID
}

func (d *Discovery) Stop() {
	discvLog.Info(fmt.Sprintf("discovery server start %s\n", d.self))

	select {
	case <-d.stop:
	default:
		close(d.stop)
	}
}

func (d *Discovery) SetNode(ip net.IP, udp, tcp uint16) {
	d.selfLock.Lock()
	defer d.selfLock.Unlock()

	if ip != nil {
		d.self.IP = ip
	}
	if udp != 0 {
		d.self.UDP = udp
	}
	if tcp != 0 {
		d.self.TCP = tcp
	}
}

func (d *Discovery) mapping() {
	out := make(chan *nat.Addr)
	go nat.Map(d.stop, "udp", int(d.self.UDP), int(d.self.UDP), "vite discovery", 0, out)

loop:
	for {
		select {
		case <-d.stop:
			break loop
		case addr := <-out:
			if addr.IsValid() {
				d.SetNode(addr.IP, uint16(addr.Port), 0)
			}
		}
	}

	d.wg.Done()
}

func New(cfg *Config) *Discovery {
	ID, err := Priv2NodeID(cfg.Priv)
	if err != nil {
		discvLog.Crit("generate NodeID from privateKey error", "error", err)
	}

	db, err := newDB(cfg.DBPath, 2, ID)
	if err != nil {
		discvLog.Crit("create p2p db", "error", err)
	}

	localAddr, _ := cfg.Conn.LocalAddr().(*net.UDPAddr)

	node := &Node{
		ID:  ID,
		IP:  localAddr.IP,
		UDP: uint16(localAddr.Port),
	}

	d := &Discovery{
		bootNodes:   cfg.BootNodes,
		self:        node,
		tab:         newTable(ID, N),
		db:          db,
		stop:        make(chan struct{}),
		refreshing:  false,
		refreshDone: make(chan struct{}),
		blockList:   block.NewCuckooSet(1000),
	}

	d.agent = &udpAgent{
		maxNeighborsOneTrip: maxNeighborsNodes,
		self:                node,
		conn:                cfg.Conn,
		priv:                cfg.Priv,
		term:                make(chan struct{}),
		running:             false,
		packetHandler:       d.HandleMsg,
	}

	return d
}
