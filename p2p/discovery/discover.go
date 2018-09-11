package discovery

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/monitor"
	"net"
	"sync"
	"time"
)

var tExpire = 24 * time.Hour
var tRefresh = 1 * time.Hour

var refreshDuration = 30 * time.Minute
var storeDuration = 5 * time.Minute
var checkInterval = 3 * time.Minute
var watingTimeout = 10 * time.Second // watingTimeout must be enough little, at least than checkInterval

func priv2ID(priv ed25519.PrivateKey) (id NodeID) {
	pub := priv.PubByte()
	copy(id[:], pub)
	return
}

type discvAgent interface {
	start()
	stop()
	ping(node *Node) error
	pong(node *Node, ack types.Hash) error
	findnode(n *Node, ID NodeID) (nodes []*Node, err error)
	sendNeighbors(n *Node, nodes []*Node) error
	want(p *packet) bool
	wait(NodeID, packetCode, waitCallback) error
}

// @section config
type Config struct {
	Priv      ed25519.PrivateKey
	DBPath    string
	BootNodes []*Node
	Addr      string // use for discover server listen, usually is local address
	PubAddr   string // announced to other peers, usually is public address
}

type Discovery struct {
	bootNodes   []*Node
	self        *Node
	agent       discvAgent
	tab         *table
	db          *nodeDB
	stop        chan struct{}
	lock        sync.RWMutex
	refreshing  bool
	refreshDone chan struct{}
}

func (d *Discovery) Start() {
	discvLog.Info("discv start")

	checkTicker := time.NewTicker(checkInterval)
	refreshTicker := time.NewTimer(tRefresh)
	storeTicker := time.NewTicker(storeDuration)

	defer checkTicker.Stop()
	defer refreshTicker.Stop()
	defer storeTicker.Stop()

	d.RefreshTable()
	go d.agent.start()

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
						d.tab.delete(n)
					} else {
						d.tab.bubble(n)
					}
				}()
			}

		case <-storeTicker.C:
			go d.tab.traverse(func(n *Node) {
				d.db.updateNode(n)
			})

		case <-d.stop:
			break loop
		}
	}

	d.agent.stop()
	d.db.close()
}

func (d *Discovery) findNode(to *Node, target NodeID) ([]*Node, error) {
	findFails := d.db.getFindNodeFails(to.ID)

	if !d.db.hasChecked(to.ID) {
		d.agent.ping(to)
		d.agent.wait(to.ID, pingCode, func(Message) bool {
			return true
		})
	}

	nodes, err := d.agent.findnode(to, target)
	if err != nil || len(nodes) == 0 {
		findFails++
		d.db.setFindNodeFails(to.ID, findFails)
		if findFails > maxFindFails {
			d.tab.delete(to)
		}
	} else if findFails > 0 {
		findFails--
		d.db.setFindNodeFails(to.ID, findFails)
	}

	for _, n := range nodes {
		d.tab.addNode(n)
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
	asked := make(map[NodeID]struct{}) // nodes has sent findnode

	resultCache := make(map[NodeID]struct{}) // all nodes of responsive neighbors, use for unique

	reply := make(chan []*Node, alpha)
	queries := 0

	result := new(neighbors)

	for {
		result.nodes = d.tab.findNeighbors(id)

		if len(result.nodes) > 0 || !refreshIfNull {
			break
		}

		d.RefreshTable()
		refreshIfNull = false
	}

	asked[d.self.ID] = struct{}{}

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
				if _, ok := resultCache[n.ID]; !ok {
					resultCache[n.ID] = struct{}{}
					result.push(n, K)
				}
			}
		}

		queries--
	}

	return result.nodes
}

func (d *Discovery) Resolve(id NodeID) *Node {
	neighbors := d.tab.findNeighbors(id)
	if len(neighbors) > 0 && neighbors[0].ID == id {
		return neighbors[0]
	}

	neighbors = d.Lookup(id)
	for _, n := range neighbors {
		if n.ID == id {
			return n
		}
	}

	return nil
}

type packet struct {
	fromID NodeID
	from   *net.UDPAddr
	code   packetCode
	hash   types.Hash
	msg    Message
}

func (d *Discovery) HandleMsg(res *packet) error {
	if res.msg.isExpired() {
		return errMsgExpired
	}
	n := &Node{
		ID:  res.fromID,
		IP:  res.from.IP,
		UDP: uint16(res.from.Port),
	}

	switch res.code {
	case pingCode:
		monitor.LogEvent("p2p/discv", "ping-receive")

		d.db.setLastPing(res.fromID, time.Now())

		d.agent.pong(n, res.hash)
		d.agent.want(res)

		if !d.db.hasChecked(res.fromID) {
			d.agent.ping(n)
		}
		d.tab.addNode(n)
	case pongCode:
		monitor.LogEvent("p2p/discv", "pong-receive")

		if !d.agent.want(res) {
			return errUnsolicitedMsg
		}
		d.db.setLastPong(res.fromID, time.Now())
	case findnodeCode:
		monitor.LogEvent("p2p/discv", "find-receive")

		if !d.db.hasChecked(res.fromID) {
			return errUnsolicitedMsg
		}
		findMsg, ok := res.msg.(*FindNode)
		if ok {
			nodes := d.tab.findNeighbors(findMsg.Target)
			d.agent.sendNeighbors(n, nodes)
		} else {
			return errUnkownMsg
		}
	case neighborsCode:
		monitor.LogEvent("p2p/discv", "neighbors-receive")

		if !d.agent.want(res) {
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
	d.lookup(d.self.ID, false)
	close(d.refreshDone)
	d.refreshDone = make(chan struct{})
}

func (d *Discovery) loadInitNodes() {
	nodes := d.db.randomNodes(seedCount, seedMaxAge)
	nodes = append(nodes, d.bootNodes...)
	for _, node := range nodes {
		d.tab.addNode(node)
	}
}

func (d *Discovery) ID() NodeID {
	return d.self.ID
}

func (d *Discovery) Stop() {
	discvLog.Info("discv stop")

	select {
	case <-d.stop:
	default:
		close(d.stop)
	}
}

func New(cfg *Config) *Discovery {
	ID := priv2ID(cfg.Priv)

	db, err := newDB(cfg.DBPath, 2, ID)
	if err != nil {
		discvLog.Crit("create p2p db", "error", err)
	}

	udpAddr, err := net.ResolveUDPAddr("udp", cfg.Addr)
	if err != nil {
		discvLog.Crit("discv listen udp", "error", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		discvLog.Crit("discv listen udp", "error", err)
	}

	pubAddr, err := net.ResolveUDPAddr("udp", cfg.PubAddr)
	realAddr := pubAddr
	if err != nil {
		realAddr = conn.LocalAddr().(*net.UDPAddr)
		discvLog.Info("discv get public address", "error", err)
	}

	node := &Node{
		ID:  ID,
		IP:  realAddr.IP,
		UDP: uint16(realAddr.Port),
	}

	discvLog.Info(node.String())

	d := &Discovery{
		bootNodes:   cfg.BootNodes,
		self:        node,
		tab:         newTable(ID, cfg.BootNodes),
		db:          db,
		stop:        make(chan struct{}),
		refreshing:  false,
		refreshDone: make(chan struct{}),
	}

	agent := &udpAgent{
		maxNeighborsOneTrip: maxNeighborsNodes,
		self:                node,
		conn:                conn,
		priv:                cfg.Priv,
		waiting:             make(chan *wait),
		res:                 make(chan *res),
		stopped:             make(chan struct{}),
		running:             false,
		packetHandler:       d.HandleMsg,
	}

	d.agent = agent

	return d
}
