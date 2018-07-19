// Package p2p implements the vite P2P network

package p2p

import (
	"time"
	"sync"
	"net"
	"errors"
	"log"
	"crypto/rand"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"reflect"
)

const (
	defaultMaxPeers = 50
	defaultDialTimeout = 10 * time.Second
	defaultMaxPendingPeers uint32 = 20
	defaultMaxActiveDail uint32 = 16
)

var errSvrHasStopped = errors.New("Server has stopped.")

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

type Config struct {
	// the counterpart publicKey is use for NodeID.
	PrivateKey ed25519.PrivateKey

	// `MaxPeers` is the maximum number of peers that can be connected.
	MaxPeers uint32

	// `MaxPassivePeersRatio` is the ratio of MaxPeers that initiate an active connection to this node.
	// the actual value is `MaxPeers / MaxPassivePeersRatio`
	MaxPassivePeersRatio uint32

	// `MaxPendingPeers` is the maximum number of peers that wait to connect.
	MaxPendingPeers uint32

	BootNodes []string

	Addr string

	// filepath of database, store former nodes
	Database string

	Name string

	NetID NetworkID
}

type peerHandler func(*Peer)

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

	// execute operations to peers by sequences.
	peersOps chan peersOperator
	// wait for operation done.
	peersOpsDone chan struct{}

	ntab *table

	ourHandshake *Handshake

	ProtoHandler peerHandler

	BootNodes []*Node
}

func NewServer(cfg *Config, handler peerHandler) (svr *Server, err error) {
	svr = &Server{
		Config: cfg,
		ProtoHandler: handler,
	}

	if svr.PrivateKey == nil {
		_, priv, err := ed25519.GenerateKey(nil)
		if err != nil {
			log.Fatal("generate self NodeID error: ", err)
		}
		svr.PrivateKey = priv
	}

	if svr.NetID == 0 {
		svr.NetID = MainNet
	}

	if svr.Addr == "" {
		svr.Addr = "localhost:8483"
	}

	if svr.Dialer == nil {
		svr.Dialer = &NodeDailer{
			&net.Dialer{
				Timeout: defaultDialTimeout,
			},
		}
	}

	// parse svr.Config.BootNodes to []*Node
	nodes := make([]*Node, 0, len(svr.Config.BootNodes))
	for _, str := range svr.Config.BootNodes {
		node, err := ParseNode(str)
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

	log.Printf(`server created.
name: %s
network: %s
db: %s
maxPeers: %d
maxPendingPeers: %d
maxActivePeers: %d
maxPassivePeers: %d`, svr.Name, svr.NetID, svr.Database, svr.MaxPeers, svr.MaxPendingPeers, svr.MaxActivePeers(), svr.MaxPassivePeers())

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
		<- svr.peersOpsDone
	case <- svr.stopped:
	}
	return
}

func (svr *Server) PeersCount() (amount int) {
	count := func(nodeTable map[NodeID]*Peer) {
		amount = len(nodeTable)
	}
	select {
	case svr.peersOps <- count:
		<- svr.peersOpsDone
	case <- svr.stopped:
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
		return errors.New("Server is already running.")
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
	id := priv2ID(svr.PrivateKey)

	svr.ourHandshake = &Handshake{
		NetID: svr.NetID,
		Name: svr.Name,
		ID: id,
	}
}

func (svr *Server) Discovery(addr *net.UDPAddr) {
	ip, err := getExtIP()
	if err != nil {
		log.Printf("discover get external ip error: %v\n", err)
	}

	cfg := &DiscvConfig{
		Priv: svr.PrivateKey,
		DBPath: svr.Database,
		BootNodes: svr.BootNodes,
		Addr: addr,
		ExternalIP: ip,
	}
	tab, err := newDiscover(cfg)

	if err != nil {
		log.Fatal(err)
	}

	if !addr.IP.IsLoopback() {
		svr.waitDown.Add(1)
		go func() {
			natMap(svr.stopped, "udp", addr.Port, addr.Port, 0, nil)
			svr.waitDown.Done()
		}()
	}

	svr.ntab = tab
}

func (svr *Server) Listen(addr *net.TCPAddr) {
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatal("tcp listen error: ", err)
	} else {
		log.Printf("tcp listen at %s\n", addr)
	}

	svr.listener = listener

	if !addr.IP.IsLoopback() {
		svr.waitDown.Add(1)
		go func() {
			natMap(svr.stopped, "tcp", addr.Port, addr.Port, 0, nil)
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
					log.Printf("tcp accept error: %v\n", err)
					continue
				}
				break
			}
			go svr.SetupConn(conn, inboundConn)
		case <- svr.stopped:
			close(svr.pendingPeers)
		}
	}
}

func (svr *Server) SetupConn(conn net.Conn, flag connFlag) error {
	log.Printf("new tcp conn from %s\n", conn.RemoteAddr())

	defer func() {
		<- svr.pendingPeers
	}()

	c := &TSConn{
		fd: conn,
		transport: svr.createTransport(conn),
		flags: flag,
	}

	log.Printf("begin handshake with %s\n", conn.RemoteAddr())

	peerhandshake, err := c.Handshake(svr.ourHandshake)

	if err != nil {
		log.Println("handshake with %s error: ", conn.RemoteAddr(), err)
		conn.Close()
		return err
	} else {
		log.Printf("handshake with %s@%s\n", peerhandshake.ID, c.fd.RemoteAddr())
	}

	c.id = peerhandshake.ID
	c.name = peerhandshake.Name

	svr.addPeer <- c

	return nil
}

func (svr *Server) CheckConn(peers map[NodeID]*Peer, passivePeersCount uint32) error {
	if uint32(len(peers)) >= svr.MaxPeers {
		return errors.New("too many peers")
	}
	if passivePeersCount >= svr.MaxPassivePeers() {
		return errors.New("too many passive peers")
	}
	return nil
}

func (svr *Server) ScheduleTask() {
	defer svr.waitDown.Done()

	dm := NewDialManager(svr.MaxActivePeers(), svr.BootNodes)
	peers := make(map[NodeID]*Peer)
	taskHasDone := make(chan Task, defaultMaxActiveDail)

	var passivePeersCount uint32 = 0
	var activeTasks []Task
	var taskQueue []Task

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
				log.Printf("perform task %s\n", reflect.TypeOf(t))
				t.Perform(svr)
				taskHasDone <- t
			}()
			activeTasks = append(activeTasks, t)
		}
		return ts[i:]
	}
	scheduleTasks := func() {
		log.Println("schedule tasks")
		taskQueue = runTasks(taskQueue)
		if uint32(len(activeTasks)) < defaultMaxActiveDail {
			newTasks := dm.CreateTasks(peers)
			log.Printf("create %d tasks\n", len(newTasks))
			if len(newTasks) > 0 {
				taskQueue = append(taskQueue, runTasks(newTasks)...)
			}
		}
	}

schedule:
	for {
		scheduleTasks()
		select {
		case <- svr.stopped:
			break schedule
		case t := <- taskHasDone:
			dm.TaskDone(t)
			delActiveTask(t)
		case c := <- svr.addPeer:
			err := svr.CheckConn(peers, passivePeersCount)
			if err == nil {
				p := NewPeer(c)
				peers[p.ID()] = p
				log.Printf("create new peer %s\n", c.id)
				go svr.runPeer(p)

				if c.is(inboundConn) {
					passivePeersCount++
				}
			} else {
				log.Printf("create new peer error: %v\n", err)
			}
		case p := <- svr.delPeer:
			delete(peers, p.ID())
			log.Printf("delete peer %s\n", p.ID())

			if p.TS.is(inboundConn) {
				passivePeersCount--
			}
		case fn := <- svr.peersOps:
			fn(peers)
			svr.peersOpsDone <- struct{}{}
		}
	}

	log.Println("out of tcp task loop")

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
		log.Println("run peer error: ", err)
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
		IP: target.IP,
		Port: int(target.Port),
	}

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
}

type dialTask struct {
	flag 			connFlag
	target 			*Node
	lastResolved 	time.Time
	duration 		time.Duration
}
func (t *dialTask) Perform(svr *Server) {
	conn, err := svr.Dialer.DailNode(t.target)
	if err != nil {
		log.Printf("dial node %s error: %v\n", t.target, err)
		return
	}

	err = svr.SetupConn(conn, dynDialedConn)
	if err != nil {
		log.Printf("setup connect to %s error: %v\n", t.target, err)
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
	maxDials uint32
	dialing map[NodeID]connFlag
	start time.Time
	bootNodes []*Node
	looking bool
	wating bool
	lookResults []*Node
}

func (dm *DialManager) CreateTasks(peers map[NodeID]*Peer) []Task {
	if dm.start.IsZero() {
		dm.start = time.Now()
	}

	var tasks []Task
	addDailTask := func(flag connFlag, n *Node) bool {
		if err := dm.checkDial(n, peers); err != nil {
			return false
		}
		dm.dialing[n.ID] = flag
		tasks = append(tasks, &dialTask{
			flag: flag,
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

	if len(peers) == 0 && len(dm.bootNodes) > 0 && dials > 0 {
		bootNode := dm.bootNodes[0]
		copy(dm.bootNodes, dm.bootNodes[1:])
		dm.bootNodes[len(dm.bootNodes) - 1] = bootNode

		if addDailTask(dynDialedConn, bootNode) {
			dials--
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
		dm.lookResults = append(dm.lookResults, t2.results...)
	case *waitTask:
		dm.wating = false
	}
}

func (dm *DialManager) checkDial(n *Node, peers map[NodeID]*Peer) error {
	_, exist := dm.dialing[n.ID];
	if exist {
		return errors.New("is dialing.")
	}
	if peers[n.ID] != nil {
		return errors.New("has connected.")
	}

	return nil
}

func NewDialManager(maxDials uint32, bootNodes []*Node) *DialManager {
	return &DialManager{
		maxDials: maxDials,
		bootNodes: copyNodes(bootNodes),	// dm will modify bootNodes
		dialing: make(map[NodeID]connFlag),
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
