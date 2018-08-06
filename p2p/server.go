// Package p2p implements the vite P2P network

package p2p

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"net"
	"path/filepath"
	"sync"
	"time"
)

var p2pServerLog = log15.New("module", "p2p/server")

const (
	defaultMaxPeers               = 50
	defaultDialTimeout            = 10 * time.Second
	defaultMaxPendingPeers uint32 = 20
	defaultMaxActiveDail   uint32 = 16
)

var errSvrHasStopped = errors.New("Server has stopped.")

var firmNodes = [...]string{
	"vnode://33e43481729850fc66cef7f42abebd8cb2f1c74f0b09a5bf03da34780a0a5606@150.109.40.224:8483",
	"vnode://7194af5b7032cb470c41b313e2675e2c3ba3377e66617247012b8d638552fb17@150.109.62.152:8483",
	"vnode://087c45631c3ec9a5dbd1189084ee40c8c4c0f36731ef2c2cb7987da421d08ba9@150.109.104.203:8483",
	"vnode://7c6a2b920764b6dddbca05bb6efa1c9bcd90d894f6e9b107f137fc496c802346@150.109.101.200:8483",
	"vnode://2840979ae06833634764c19e72e6edbf39595ff268f558afb16af99895aba3d8@150.109.105.192:8483",
	"vnode://298a693584e4fceebb3f7abab2c25e7a6c6b911f15c12362b23144dd72822c02@119.28.224.63:8483",
	"vnode://75f962a81a6d52d9dcc830ff7dc2f21c424eeed4a0f2e5bab8f39f44df833153@150.109.104.186:8483",
	"vnode://491d7a992cad4ba82ea10ad0f5da86a2e40f4068918bd50d8faae1f1b69d8510@150.109.42.191:8483",
	"vnode://c3debe5fc3f8839c4351834d454a0940872f973ad0d332811f0d9e84953cfdc2@150.109.104.12:8483",
	"vnode://ba351c5df80eea3d78561507e40b3160ade4daf20571c5a011ca358b81d630f7@150.109.104.53:8483",
	"vnode://a1d437148c48a44b709b0a0c1e99c847bb4450384a6eea899e35e78a1e54c92b@150.109.103.236:8483",
	"vnode://657f6706b95005a33219ae26944e6db15edfe8425b46fa73fb6cae57707b4403@150.109.32.116:8483",
	"vnode://20fd0ec94071ea99019d3c18a5311993b8fa91920b36803e5558783ca346cec1@150.109.102.180:8483",
	"vnode://8669feb2fdeeedfbbfe8754050c0bd211a425e3b999d44856867d504cf13243e@150.109.59.74:8483",
	"vnode://802d8033f6adea154e5fa8b356a45213e8a4e5e85e54da19688cb0ad3520db2b@150.109.105.23:8483",
	"vnode://180af9c90f8444a452a9fb53f3a1975048e9b23ec492636190f9741ed3888a62@150.109.105.145:8483",
	"vnode://6c8ce60b87199d46f7524d7da5a06759c054b9192818d0e8b211768412efec51@150.109.105.26:8483",
	"vnode://3b3e7164827eb339548b9c895fc56acee7980a0044b5bef2e6f465e030944e40@150.109.62.157:8483",
	"vnode://de1f3a591b551591fb7d20e478da0371def3d62726b90ffca6932e38a25ebe84@150.109.38.29:8483",
	"vnode://9df2e11399398176fa58638592cf1b2e0e804ae92ac55f09905618fdb239c03c@150.109.40.169:8483",
}

// type of conn
type connFlag int

const (
	dynDialedConn connFlag = 1 << iota
	inboundConn
)

func (f connFlag) is(f2 connFlag) bool {
	return (f & f2) != 0
}

type TSConn struct {
	fd net.Conn
	transport
	flags connFlag
	cont  chan error
	id    NodeID
	name  string
}

func (c *TSConn) is(flag connFlag) bool {
	return c.flags.is(flag)
}

type peerHandler func(*Peer)

type Config struct {
	*config.P2P

	NetID NetworkID

	Database string

	PrivateKey ed25519.PrivateKey
	// use for NodeID
	PublicKey ed25519.PublicKey
}

type Server struct {
	*Config

	running bool

	lock sync.Mutex

	// Wait for shutdown clean jobs done.
	waitDown sync.WaitGroup

	Dialer *NodeDailer

	// TCP listener
	listener *net.TCPListener

	createTransport func(conn net.Conn) transport

	// Indicate whether the server has stopped. If stopped, zero-value can be read from this channel.
	stopped chan struct{}

	pendingPeers chan struct{}

	addPeer chan *TSConn

	delPeer chan *Peer

	blocknode chan *Node

	// execute operations to peers by sequences.
	peersOps chan peersOperator
	// wait for operation done.
	peersOpsDone chan struct{}

	ntab *table

	ourHandshake *Handshake

	ProtoHandler peerHandler

	BootNodes []*Node

	log log15.Logger
}

func NewServer(cfg *config.P2P, handler peerHandler) (svr *Server, err error) {
	config := &Config{
		P2P: cfg,
	}
	config.NetID = NetworkID(cfg.NetID)

	if config.Name != "" && config.Sig != "" {
		config.PublicKey = pickPub(config.Name, config.Sig)
		fmt.Println()
	}

	if cfg.PublicKey != "" {
		pub, err := hex.DecodeString(cfg.PublicKey)
		if err == nil {
			config.PublicKey = pub
		} else {
			p2pServerLog.Info("publicKey decode", "err", err)
		}
	}

	if cfg.PrivateKey != "" {
		priv, err := hex.DecodeString(cfg.PrivateKey)
		if err == nil {
			config.PrivateKey = priv
		} else {
			p2pServerLog.Info("privateKey decode", "err", err)
		}
	}

	if config.PrivateKey == nil && config.PublicKey == nil {
		pub, priv, err := ed25519.GenerateKey(nil)
		if err != nil {
			p2pServerLog.Crit("generate self NodeID", "err", err)
		}
		config.PrivateKey = priv
		config.PublicKey = pub
	}

	if config.NetID == 0 {
		config.NetID = TestNet
	}

	if config.Addr == "" {
		config.Addr = "0.0.0.0:8483"
	}

	if config.Database == "" {
		config.Database = filepath.Join(config.Datadir, "p2p")
	}

	// after default config
	svr = &Server{
		Config:       config,
		ProtoHandler: handler,
		log:          p2pServerLog.New("module", "p2p/server"),
	}

	if svr.Dialer == nil {
		svr.Dialer = &NodeDailer{
			&net.Dialer{
				Timeout: defaultDialTimeout,
			},
		}
	}

	// parse svr.Config.BootNodes to []*Node
	nodes := make([]*Node, 0, len(svr.Config.BootNodes)+len(firmNodes))
	for _, str := range svr.Config.BootNodes {
		node, err := ParseNode(str)
		if err == nil {
			nodes = append(nodes, node)
		}
	}
	for _, fnode := range firmNodes {
		node, err := ParseNode(fnode)
		if err == nil {
			nodes = append(nodes, node)
		}
	}

	svr.BootNodes = nodes

	svr.stopped = make(chan struct{})
	svr.addPeer = make(chan *TSConn)
	svr.delPeer = make(chan *Peer)
	svr.peersOps = make(chan peersOperator)
	svr.peersOpsDone = make(chan struct{})
	svr.blocknode = make(chan *Node)

	if svr.MaxPeers == 0 {
		svr.MaxPeers = defaultMaxPeers
	}

	if svr.MaxPendingPeers == 0 {
		svr.MaxPendingPeers = defaultMaxPendingPeers
	}
	maxPendingPeers := svr.MaxPendingPeers

	svr.pendingPeers = make(chan struct{}, maxPendingPeers)

	if svr.MaxPassivePeersRatio == 0 {
		svr.MaxPassivePeersRatio = 3
	}

	if svr.createTransport == nil {
		svr.createTransport = NewPBTS
	}

	svr.SetHandshake()

	svr.log.Info("server created", "name", svr.Name, "network", svr.NetID.String(), "db", svr.Database, "maxPeers", svr.MaxPeers, "maxPendingPeers", svr.MaxPendingPeers, "maxActivePeers", svr.MaxActivePeers(), "maxPassivePeers", svr.MaxPassivePeers())

	return svr, nil
}

type peersOperator func(nodeTable map[NodeID]*Peer)

func (svr *Server) Peers() (peers []*Peer) {
	map2slice := func(nodeTable map[NodeID]*Peer) {
		for _, node := range nodeTable {
			peers = append(peers, node)
		}
	}
	select {
	case svr.peersOps <- map2slice:
		<-svr.peersOpsDone
	case <-svr.stopped:
	}
	return
}

func (svr *Server) PeersCount() (amount int) {
	count := func(nodeTable map[NodeID]*Peer) {
		amount = len(nodeTable)
	}
	select {
	case svr.peersOps <- count:
		<-svr.peersOpsDone
	case <-svr.stopped:
	}
	return
}

func (svr *Server) Available() bool {
	count := svr.PeersCount()
	return count > 0
}

func (svr *Server) MaxActivePeers() uint32 {
	return svr.MaxPeers - svr.MaxPassivePeers()
}

func (svr *Server) MaxPassivePeers() uint32 {
	return svr.MaxPeers / svr.MaxPassivePeersRatio
}

func (svr *Server) Start() error {
	svr.lock.Lock()
	defer svr.lock.Unlock()

	if svr.running {
		return errors.New("server is already running")
	}
	svr.running = true

	// udp discover
	udpAddr, err := net.ResolveUDPAddr("udp", svr.Addr)
	if err != nil {
		return err
	}
	svr.Discovery(udpAddr)

	// tcp listener
	tcpAddr, err := net.ResolveTCPAddr("tcp", svr.Addr)
	if err != nil {
		return err
	}
	go svr.Listen(tcpAddr)

	// task loop
	svr.waitDown.Add(1)
	go svr.ScheduleTask()

	return nil
}

func (svr *Server) SetHandshake() {
	var id NodeID

	if svr.PublicKey == nil {
		id = priv2ID(svr.PrivateKey)
	} else {
		copy(id[:], svr.PublicKey)
	}

	svr.ourHandshake = &Handshake{
		NetID:   svr.NetID,
		Name:    svr.Name,
		ID:      id,
		Version: Version,
	}
}

func (svr *Server) Discovery(addr *net.UDPAddr) {
	cfg := &DiscvConfig{
		Priv:      svr.PrivateKey,
		Pub:       svr.PublicKey,
		DBPath:    svr.Database,
		BootNodes: svr.BootNodes,
		Addr:      addr,
	}

	tab, laddr, err := newDiscover(cfg)

	if err != nil {
		svr.log.Crit("udp discv", "err", err)
	}

	if !laddr.IP.IsLoopback() {
		svr.waitDown.Add(1)
		go func() {
			natMap(svr.stopped, "udp", laddr.Port, laddr.Port, 0)
			svr.waitDown.Done()
		}()
	}

	svr.ntab = tab
}

func (svr *Server) Listen(addr *net.TCPAddr) {
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		svr.log.Crit("tcp listen", "err", err)
	} else {
		svr.log.Info("tcp listening", "addr", addr.String())
	}

	svr.listener = listener

	realaddr := listener.Addr().(*net.TCPAddr)
	if !realaddr.IP.IsLoopback() {
		svr.waitDown.Add(1)
		go func() {
			natMap(svr.stopped, "tcp", addr.Port, addr.Port, 0)
			svr.waitDown.Done()
		}()
	}

	svr.waitDown.Add(1)
	go svr.handleConn()
}

func (svr *Server) handleConn() {
	defer svr.waitDown.Done()

	var conn net.Conn
	var err error

	for {
		select {
		case svr.pendingPeers <- struct{}{}:
			for {
				conn, err = svr.listener.Accept()

				if err != nil {
					svr.log.Error("tcp accept error", "error", err)
					continue
				}
				break
			}
			go svr.SetupConn(conn, inboundConn)
		case <-svr.stopped:
			close(svr.pendingPeers)
		}
	}
}

func (svr *Server) SetupConn(conn net.Conn, flag connFlag) error {
	svr.log.Info("new tcp conn", "from", conn.RemoteAddr().String())

	defer func() {
		<-svr.pendingPeers
	}()

	c := &TSConn{
		fd:        conn,
		transport: svr.createTransport(conn),
		flags:     flag,
	}

	svr.log.Info("begin handshake", "with", conn.RemoteAddr().String())

	peerhandshake, err := c.Handshake(svr.ourHandshake)

	if err != nil {
		svr.log.Error("handshake error", "with", conn.RemoteAddr(), "error", err)
		conn.Close()
		return err
	} else {
		svr.log.Info("handshake done", "with", peerhandshake.ID.String()+"@"+c.fd.RemoteAddr().String())
	}

	c.id = peerhandshake.ID
	c.name = peerhandshake.Name

	svr.addPeer <- c

	return nil
}

func (svr *Server) CheckConn(peers map[NodeID]*Peer, c *TSConn, passivePeersCount uint32) error {
	if uint32(len(peers)) >= svr.MaxPeers {
		return DiscTooManyPeers
	}
	if passivePeersCount >= svr.MaxPassivePeers() {
		return DiscTooManyPassivePeers
	}
	if peers[c.id] != nil {
		return DiscAlreadyConnected
	}
	if c.id == svr.ntab.self.ID {
		return DiscSelf
	}
	return nil
}

type blockNode struct {
	node      *Node
	blockTime time.Time
}

var defaultBlockTimeout = 2 * time.Minute

func (svr *Server) ScheduleTask() {
	defer svr.waitDown.Done()

	dm := NewDialManager(svr.ntab, svr.MaxActivePeers(), svr.BootNodes)
	peers := make(map[NodeID]*Peer)
	taskHasDone := make(chan Task, defaultMaxActiveDail)

	var passivePeersCount uint32 = 0
	var activeTasks []Task
	var taskQueue []Task

	blocknodes := make(map[NodeID]*blockNode)
	cleanBlockTicker := time.NewTicker(defaultBlockTimeout)
	defer cleanBlockTicker.Stop()

	delActiveTask := func(t Task) {
		for i, at := range activeTasks {
			if at == t {
				activeTasks = append(activeTasks[:i], activeTasks[i+1:]...)
			}
		}
	}
	runTasks := func(ts []Task) (rest []Task) {
		i := 0
		for ; uint32(len(activeTasks)) < defaultMaxActiveDail && i < len(ts); i++ {
			t := ts[i]
			go func() {
				t.Perform(svr)
				taskHasDone <- t
			}()
			activeTasks = append(activeTasks, t)
		}
		return ts[i:]
	}
	scheduleTasks := func() {
		taskQueue = runTasks(taskQueue)
		if uint32(len(activeTasks)) < defaultMaxActiveDail {
			newTasks := dm.CreateTasks(peers, blocknodes)
			if len(newTasks) > 0 {
				taskQueue = append(taskQueue, runTasks(newTasks)...)
			}
		}
	}

schedule:
	for {
		scheduleTasks()

		select {
		case <-svr.stopped:
			break schedule
		case t := <-taskHasDone:
			dm.TaskDone(t)
			delActiveTask(t)
		case c := <-svr.addPeer:
			err := svr.CheckConn(peers, c, passivePeersCount)
			if err == nil {
				p := NewPeer(c)
				peers[p.ID()] = p
				svr.log.Info("create new peer", "ID", c.id.String())
				go svr.runPeer(p)

				if c.is(inboundConn) {
					passivePeersCount++
				}
			} else {
				c.Close(err)
				svr.log.Error("create new peer error", "error", err)
			}
		case p := <-svr.delPeer:
			delete(peers, p.ID())
			svr.log.Info("delete peer", "ID", p.ID().String())

			if p.TS.is(inboundConn) {
				passivePeersCount--
			}
		case fn := <-svr.peersOps:
			fn(peers)
			svr.peersOpsDone <- struct{}{}
		case node := <-svr.blocknode:
			svr.log.Info("block node", "node", node.String())
			blocknodes[node.ID] = &blockNode{
				node,
				time.Now(),
			}
		case <-cleanBlockTicker.C:
			now := time.Now()
			for id, blockNode := range blocknodes {
				if now.Sub(blockNode.blockTime) > defaultBlockTimeout {
					delete(blocknodes, id)
				}
			}
		}
	}

	svr.log.Info("out of tcp task loop")

	if svr.ntab != nil {
		svr.ntab.stop()
	}

	for _, p := range peers {
		p.Disconnect(DiscQuitting)
	}

	// wait for peers work down.
	for p := range svr.delPeer {
		delete(peers, p.ID())
	}
}

func (svr *Server) runPeer(p *Peer) {
	err := p.run(svr.ProtoHandler)
	if err != nil {
		svr.log.Error("run peer error", "error", err)
	}
	svr.delPeer <- p
}

func (svr *Server) Stop() {
	svr.lock.Lock()
	defer svr.lock.Unlock()

	if !svr.running {
		return
	}

	svr.running = false

	if svr.listener != nil {
		svr.listener.Close()
	}

	if svr.ntab != nil {
		svr.ntab.stop()
	}

	close(svr.stopped)
	svr.waitDown.Wait()
}

// @section Dialer
type NodeDailer struct {
	*net.Dialer
}

func (d *NodeDailer) DailNode(target *Node) (net.Conn, error) {
	addr := net.TCPAddr{
		IP:   target.IP,
		Port: int(target.Port),
	}
	p2pServerLog.Info("tcp dial", "node", target)
	return d.Dialer.Dial("tcp", addr.String())
}

// @section Task
type Task interface {
	Perform(svr *Server)
}

type discoverTask struct {
	results []*Node
}

func (t *discoverTask) Perform(svr *Server) {
	var target NodeID
	rand.Read(target[:])
	t.results = svr.ntab.lookup(target)
	p2pServerLog.Info(fmt.Sprintf("discv tab lookup %s %d nodes\n", target, len(t.results)))
}

type dialTask struct {
	flag         connFlag
	target       *Node
	lastResolved time.Time
	duration     time.Duration
}

func (t *dialTask) Perform(svr *Server) {
	if t.target.ID == svr.ntab.self.ID {
		return
	}

	conn, err := svr.Dialer.DailNode(t.target)
	if err != nil {
		p2pServerLog.Info(fmt.Sprintf("tcp dial node %s error: %v\n", t.target, err))
		svr.blocknode <- t.target
		return
	}

	err = svr.SetupConn(conn, dynDialedConn)
	if err != nil {
		p2pServerLog.Info(fmt.Sprintf("setup connect to %s error: %v\n", t.target, err))
		svr.blocknode <- t.target
	}
}

type waitTask struct {
	Duration time.Duration
}

func (t *waitTask) Perform(svr *Server) {
	time.Sleep(t.Duration)
}

// @section DialManager
type DialManager struct {
	maxDials    uint32
	dialing     map[NodeID]connFlag
	start       time.Time
	bootNodes   []*Node
	looking     bool
	wating      bool
	lookResults []*Node
	ntab        *table
}

func (dm *DialManager) CreateTasks(peers map[NodeID]*Peer, blockList map[NodeID]*blockNode) []Task {
	if dm.start.IsZero() {
		dm.start = time.Now()
	}

	var tasks []Task
	addDailTask := func(flag connFlag, n *Node) bool {
		if _, ok := blockList[n.ID]; ok {
			return false
		}

		if err := dm.checkDial(n, peers); err != nil {
			return false
		}

		dm.dialing[n.ID] = flag
		tasks = append(tasks, &dialTask{
			flag:   flag,
			target: n,
		})
		return true
	}

	dials := dm.maxDials
	for _, p := range peers {
		if p.TS.is(dynDialedConn) {
			dials--
		}
	}
	for _, f := range dm.dialing {
		if f.is(dynDialedConn) {
			dials--
		}
	}

	// bootNodes
	for i := 0; i < len(dm.bootNodes) && dials > 0; i++ {
		bootNode := dm.bootNodes[i]

		if addDailTask(dynDialedConn, bootNode) {
			dials--
		}
	}

	// randomNodes from table
	randomCandidates := dials / 2
	if randomCandidates > 0 {
		randomNodes := make([]*Node, randomCandidates)
		n := dm.ntab.readRandomNodes(randomNodes)
		for i := 0; i < n; i++ {
			if addDailTask(dynDialedConn, randomNodes[i]) {
				dials--
			}
		}
	}

	resultIndex := 0
	for ; resultIndex < len(dm.lookResults) && dials > 0; resultIndex++ {
		if addDailTask(dynDialedConn, dm.lookResults[resultIndex]) {
			dials--
		}
	}
	dm.lookResults = dm.lookResults[resultIndex:]

	if len(dm.lookResults) == 0 && !dm.looking {
		tasks = append(tasks, &discoverTask{})
		dm.looking = true
	}

	if len(tasks) == 0 && !dm.wating {
		tasks = append(tasks, &waitTask{
			Duration: 3 * time.Minute,
		})
		dm.wating = true
	}

	return tasks
}

func (dm *DialManager) TaskDone(t Task) {
	switch t2 := t.(type) {
	case *dialTask:
		delete(dm.dialing, t2.target.ID)
	case *discoverTask:
		dm.looking = false

		self := dm.ntab.self.ID
		for _, node := range t2.results {
			if self != node.ID {
				dm.lookResults = append(dm.lookResults, node)
			}
		}
	case *waitTask:
		dm.wating = false
	}
}

func (dm *DialManager) checkDial(n *Node, peers map[NodeID]*Peer) error {
	_, exist := dm.dialing[n.ID]
	if exist {
		return fmt.Errorf("%s is dialing", n)
	}
	if peers[n.ID] != nil {
		return fmt.Errorf("%s has connected", n)
	}
	if n.ID == dm.ntab.self.ID {
		return errors.New("self node")
	}

	return nil
}

func NewDialManager(ntab *table, maxDials uint32, bootNodes []*Node) *DialManager {
	return &DialManager{
		maxDials:  maxDials,
		bootNodes: copyNodes(bootNodes), // dm will modify bootNodes
		dialing:   make(map[NodeID]connFlag),
		ntab:      ntab,
	}
}

func copyNodes(nodes []*Node) []*Node {
	cpnodes := make([]*Node, len((nodes)))

	for i, nodep := range nodes {
		node := *nodep
		cpnodes[i] = &node
	}

	return cpnodes
}
