package discovery

import (
	"crypto/rand"
	"fmt"
	mrand "math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/p2p/list"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p/network"
)

const seedCount = 50
const seedMaxAge = 7 * 24 * time.Hour

var tExpire = 24 * time.Hour         // nodes have been ping-pong checked during this time period are considered valid
var tRefresh = 1 * time.Hour         // refresh the node table at tRefresh intervals
var storeInterval = 5 * time.Minute  // store nodes in table to db at storeDuration intervals
var checkInterval = 10 * time.Second // check the oldest node in table at checkInterval intervals
var stayInTable = 5 * time.Minute    // minimal duration node stay in table can be store in db
var dbCleanInterval = time.Hour
var findInterval = 30 * time.Second

// Priv2NodeID got the corresponding NodeID from ed25519.PeerKey
func Priv2NodeID(priv ed25519.PrivateKey) (NodeID, error) {
	pub := priv.PubByte()
	return Bytes2NodeID(pub)
}

// Config is the essential configuration to create a Discovery implementation
type Config struct {
	PeerKey   ed25519.PrivateKey
	DBPath    string
	BootNodes []*Node
	Addr      string
	NetID     network.ID
	Self      *Node
}

// Discovery is the interface to discovery other node
type Discovery interface {
	Start() error
	Stop()
	SubNodes(ch chan<- *Node, near bool)
	UnSubNodes(ch chan<- *Node)
	Mark(id NodeID, lifetime int64)
	UnMark(id NodeID)
	Block(id NodeID, ip net.IP)
	More(ch chan<- *Node)
	Nodes() []string
}

type discovery struct {
	*Config
	*table
	agent    *agent
	db       *nodeDB
	term     chan struct{}
	pingChan chan *Node
	findChan chan *Node
	finding  sync.Map
	looking  int32 // is looking self
	wg       sync.WaitGroup
	log      log15.Logger
}

func (d *discovery) More(ch chan<- *Node) {
	nodes := d.table.near()

	go func() {
		for _, node := range nodes {
			ch <- node
		}

		if len(nodes) == 0 {
			d.init()
		}
	}()
}
func (d *discovery) Nodes() (nodes []string) {
	d.table.m.Range(func(key, value interface{}) bool {
		if node := value.(*Node); node != nil {
			nodes = append(nodes, node.String())
		}
		return true
	})

	return nodes
}

// New create a Discovery implementation
func New(cfg *Config) Discovery {
	d := &discovery{
		Config:   cfg,
		table:    newTable(cfg.Self.ID, cfg.NetID),
		pingChan: make(chan *Node, 10),
		findChan: make(chan *Node, 5),
		log:      log15.New("module", "p2p/discv"),
	}

	d.agent = &agent{
		self:    cfg.Self,
		peerKey: cfg.PeerKey,
		handler: d.handle,
		pool:    newWtPool(),
		log:     d.log,
	}

	return d
}

func (d *discovery) Start() (err error) {
	d.log.Info(fmt.Sprintf("discovery %s start", d.Self.ID))

	udpAddr, err := net.ResolveUDPAddr("udp", d.Addr)
	if err != nil {
		return
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return
	}
	d.agent.conn = conn

	db, err := newDB(d.Config.DBPath, 3, d.Self.ID)
	if err != nil {
		d.log.Error(fmt.Sprintf("create p2p db error: %v", err))
	}
	d.db = db

	d.agent.start()

	d.term = make(chan struct{})

	d.wg.Add(1)
	common.Go(d.tableLoop)

	d.wg.Add(1)
	common.Go(d.pingLoop)

	d.wg.Add(1)
	common.Go(d.findLoop)

	return
}

func (d *discovery) Stop() {
	if d.term == nil {
		return
	}

	select {
	case <-d.term:
	default:
		d.log.Warn(fmt.Sprintf("stop discovery %s", d.Self.ID))

		close(d.term)
		d.db.close()
		d.agent.stop()
		d.wg.Wait()

		d.log.Warn(fmt.Sprintf("discovery %s stopped", d.Self.ID))
	}
}

func (d *discovery) Block(ID NodeID, IP net.IP) {
	// todo
}

func (d *discovery) Mark(id NodeID, lifetime int64) {
	if node := d.table.resolveById(id); node != nil {
		node.mark = lifetime
	}
}

func (d *discovery) UnMark(id NodeID) {
	if node := d.table.resolveById(id); node != nil {
		node.mark = 0
	}
}

func (d *discovery) pingLoop() {
	defer d.wg.Done()

	var pending sync.Map

	const alpha = 10
	tickets := make(chan struct{}, 10)
	for i := 0; i < alpha; i++ {
		tickets <- struct{}{}
	}

	run := func(node *Node) {
		node.lastPing = time.Now()
		done := make(chan bool, 1)
		d.agent.ping(node.ID, node.UDPAddr(), done)

		if <-done {
			d.bubbleOrAdd(node)
		} else {
			d.remove(node)
		}

		tickets <- struct{}{}
		addr := node.UDPAddr().String()
		pending.Delete(addr)
	}

	wait := list.New()

	do := func(node *Node) {
		select {
		case <-tickets:
			go run(node)
		default:
			wait.Append(node)
		}
	}

	for {
		select {
		case <-d.term:
			return
		case node := <-d.pingChan:
			addr := node.UDPAddr().String()
			if _, loaded := pending.LoadOrStore(addr, struct{}{}); loaded {
				continue
			} else {
				do(node)
			}
		default:
			if e := wait.Shift(); e != nil {
				do(e.(*Node))
			} else {
				time.Sleep(time.Second)
			}
		}
	}
}

func (d *discovery) findLoop() {
	defer d.wg.Done()

	var pending sync.Map

	const alpha = 10
	tickets := make(chan struct{}, 10)
	for i := 0; i < alpha; i++ {
		tickets <- struct{}{}
	}

	run := func(node *Node) {
		node.lastFind = time.Now()
		ch := make(chan []*Node, 1)

		id := d.Self.ID
		if mrand.Intn(10) < 5 {
			rand.Read(id[:])
		}
		d.agent.findnode(node.ID, node.UDPAddr(), id, maxNeighborsOneTrip, ch)

		nodes := <-ch
		tickets <- struct{}{}
		addr := node.UDPAddr().String()
		pending.Delete(addr)
		d.log.Info(fmt.Sprintf("got %d neighbors from %s", len(nodes), node.UDPAddr()))
	}

	wait := list.New()

	do := func(node *Node) {
		select {
		case <-tickets:
			go run(node)
		default:
			wait.Append(node)
		}
	}

	for {
		select {
		case <-d.term:
			return
		case node := <-d.findChan:
			addr := node.UDPAddr().String()
			if _, loaded := pending.LoadOrStore(addr, struct{}{}); loaded {
				continue
			} else {
				do(node)
			}
		default:
			if e := wait.Shift(); e != nil {
				do(e.(*Node))
			} else {
				time.Sleep(time.Second)
			}
		}
	}
}

func (d *discovery) tableLoop() {
	defer d.wg.Done()

	d.init()

	checkTicker := time.NewTicker(checkInterval)
	refreshTicker := time.NewTimer(tRefresh)
	storeTicker := time.NewTicker(storeInterval)
	dbTicker := time.NewTicker(dbCleanInterval)
	findTicker := time.NewTicker(findInterval)

	defer checkTicker.Stop()
	defer refreshTicker.Stop()
	defer storeTicker.Stop()
	defer dbTicker.Stop()
	defer findTicker.Stop()

Loop:
	for {
		select {
		case <-d.term:
			break Loop
		case <-checkTicker.C:
			nodes := d.pickOldest()
			for _, node := range nodes {
				d.pingNode(node)
			}

		case <-findTicker.C:
			if d.size() == 0 {
				d.init()
			} else if d.table.needMore() {
				d.lookSelf()
			}

		case <-storeTicker.C:
			d.table.store(d.db)

		case <-dbTicker.C:
			d.db.cleanStaleNodes()
		}
	}

	d.table.store(d.db)
}

func (d *discovery) handle(res *packet) {
	node := &Node{
		ID:  res.fromID,
		IP:  res.from.IP,
		UDP: uint16(res.from.Port),
	}
	if ping, ok := res.msg.(*Ping); ok {
		node.Net = ping.Net
		node.Ext = ping.Ext
	}

	d.seeNode(node)

	switch res.code {
	case pingCode:
		monitor.LogEvent("p2p/discv", "ping-receive")
		d.agent.pong(res.from, res.hash)

	case pongCode:
		monitor.LogEvent("p2p/discv", "pong-receive")

	case findnodeCode:
		monitor.LogEvent("p2p/discv", "find-receive")

		findMsg := res.msg.(*FindNode)
		total := findMsg.N
		if total == 0 {
			total = maxNeighborsOneTrip
		}
		nodes := d.table.findNeighbors(findMsg.Target, int(total))
		d.agent.sendNeighbors(res.from, nodes)

	case neighborsCode:
		monitor.LogEvent("p2p/discv", "neighbors-receive")

		nodes := res.msg.(*Neighbors).Nodes

		for _, n := range nodes {
			d.seeNode(n)
		}
	}
}

func (d *discovery) seeNode(n *Node) {
	if n.ID == d.Self.ID {
		return
	}

	if node := d.table.resolve(n.UDPAddr().String()); node != nil {
		if node.ID != n.ID {
			d.pingNode(n)
		} else {
			node.Update(n)
			d.bubble(n.ID)

			if node.shouldFind() && d.table.needMore() {
				d.findNode(n)
			}
		}
	} else {
		d.pingNode(n)
	}
}

func (d *discovery) pingNode(n *Node) {
	select {
	case <-d.term:
	default:
		d.pingChan <- n
	}
}
func (d *discovery) findNode(n *Node) {
	select {
	case <-d.term:
	default:
		d.findChan <- n
	}
}

func (d *discovery) init() {
	nodes := d.db.randomNodes(seedCount, seedMaxAge) // get random nodes from db
	d.log.Info(fmt.Sprintf("got %d nodes from db", len(nodes)))

	notified := 0
	for _, node := range nodes {
		if node.shouldPing() {
			d.pingNode(node)
		} else {
			d.addNode(node)
			if node.mark > 0 && notified < 5 {
				notified++
				d.notifyAll(node)
			}
		}
	}

	// send findnode to bootnode directly, bypass ping-pong check
	for _, node := range d.BootNodes {
		d.table.addNode(node)
	}

	if len(nodes)+len(d.BootNodes) == 0 {
		d.log.Error("no bootNodes")
		return
	}

	nodes = d.lookSelf()

	if len(nodes) == 0 {
		ids := mrand.Perm(len(d.BootNodes))
		total := len(ids)
		if total > 3 {
			total = 3
		}

		for i := 0; i < total; i++ {
			idx := ids[i]
			d.notifyAll(d.BootNodes[idx])
		}
	}
}

func (d *discovery) lookup(target NodeID) []*Node {
	if atomic.CompareAndSwapInt32(&d.looking, 0, 1) {
		goto Look
	} else {
		return nil
	}

Look:
	defer atomic.StoreInt32(&d.looking, 0)

	const total = 3 * maxNeighborsOneTrip
	var result = neighbors{
		pivot: target,
	}

	result.nodes = d.table.findNeighbors(target, total)

	if len(result.nodes) == 0 {
		return nil
	}

	asked := make(map[NodeID]struct{}) // nodes has sent findnode message
	asked[d.Self.ID] = struct{}{}

	// all nodes of responsive neighbors, use for filter to ensure the same node pushed once
	seen := make(map[NodeID]struct{})

	const alpha = 10
	reply := make(chan []*Node, alpha)
	queries := 0

Loop:
	for {
		for i := 0; i < len(result.nodes) && queries < alpha; i++ {
			n := result.nodes[i]
			if _, ok := asked[n.ID]; !ok && n.shouldFind() {
				asked[n.ID] = struct{}{}
				n.lastFind = time.Now()
				d.agent.findnode(n.ID, n.UDPAddr(), target, total, reply)
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
				if n != nil && n.ID != target {
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

func (d *discovery) lookSelf() (nodes []*Node) {
	nodes = d.lookup(d.Self.ID)
	if len(nodes) == 0 {
		return
	}

	for _, node := range nodes {
		d.table.addNode(node)
		d.notifyAll(node)
	}

	return
}
