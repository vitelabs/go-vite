// Package p2p implements the vite P2P network

package p2p

import (
	"time"
	"sync"
	"net"
	"errors"
	"log"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"crypto/rand"
)

const (
	defaultDialTimeout = 10 * time.Second
	defaultMaxPendingPeers uint32 = 30
	defaultMaxActiveDail uint32 = 16
)

var errSvrHasStopped = errors.New("Server has stopped.")

// type of conn
type connFlag int

const (
	dynDialedConn connFlag = 1 << iota
	staticDialedConn
	inboundConn
	trustedConn
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
	// mandatory, `PrivateKey` must be set
	PrivateKey ed25519.PrivateKey

	// `MaxPeers` is the maximum number of peers that can be connected.
	MaxPeers uint32

	// `MaxPassivePeersRatio` is the ratio of MaxPeers that initiate an active connection to this node.
	// the actual value is `MaxPeers / MaxPassivePeersRatio`
	MaxPassivePeersRatio uint32

	// `MaxPendingPeers` is the maximum number of peers that wait to connect.
	MaxPendingPeers uint32

	BootNodes []*Node

	Addr string

	// filepath of database, store former nodes
	Database string

	Name string
}

type Server struct {
	Config

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

	ourHandshake *protoHandshake
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

func (svr *Server) MaxActivePeers() uint32 {
	return svr.MaxPeers - svr.MaxPassivePeers()
}

func (svr *Server) MaxPassivePeers() uint32 {
	if svr.MaxPassivePeersRatio == 0 {
		svr.MaxPassivePeersRatio = 3
	}
	return svr.MaxPeers / svr.MaxPassivePeersRatio
}

func (svr *Server) Start() error {
	svr.lock.Lock()
	defer svr.lock.Unlock()

	if svr.running {
		return errors.New("Server is already running.")
	}
	svr.running = true

	if svr.PrivateKey == nil {
		return errors.New("Server.PrivateKey must set, but get nil.")
	}

	if svr.Dialer == nil {
		svr.Dialer = &NodeDailer{
			&net.Dialer{
				Timeout: defaultDialTimeout,
			},
		}
	}

	svr.stopped = make(chan struct{})
	svr.addPeer = make(chan *TSConn)
	svr.delPeer = make(chan *Peer)
	svr.peersOps = make(chan peersOperator)
	svr.peersOpsDone = make(chan struct{})

	maxPendingPeers := defaultMaxPendingPeers
	if svr.MaxPendingPeers > 0 {
		maxPendingPeers = svr.MaxPendingPeers
	}

	svr.pendingPeers = make(chan struct{}, maxPendingPeers)

	if svr.createTransport == nil {
		svr.createTransport = NewPBTS
	}

	svr.SetHandleshake()

	// udp discover
	udpAddr, err := net.ResolveUDPAddr("udp", svr.Addr)
	if err != nil {
		return err
	}
	go svr.Discovery(udpAddr)

	// tcp listener
	tcpAddr, err := net.ResolveTCPAddr("tcp", svr.Addr)
	if err != nil {
		return err
	}
	go svr.Listen(tcpAddr)

	// task
	svr.waitDown.Add(1)
	go svr.ManageTask()

	return nil
}

func (svr *Server) SetHandleshake() {
	id := priv2ID(svr.PrivateKey)

	svr.ourHandshake = &protoHandshake{
		Name: svr.Name,
		ID: id,
	}
}

func (svr *Server) Discovery(addr *net.UDPAddr) {
	ip, err := getExtIP()
	if err != nil {
		log.Fatal(err)
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
		go natMap(svr.stopped, "udp", addr.Port, addr.Port, 0, nil)
	}

	svr.ntab = tab
}

func (svr *Server) Listen(addr *net.TCPAddr) {
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatal("tcp listen error: ", err)
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

	for {
		var conn net.Conn
		var err error

		select {
		case svr.pendingPeers <- struct{}{}:
			for {
				conn, err = svr.listener.Accept()
				if err != nil {
					log.Fatal(err)
				} else {
					break
				}
			}

			go svr.SetupConn(conn, inboundConn)
		case <- svr.stopped:
			close(svr.pendingPeers)
		}
	}
}

func (svr *Server) SetupConn(conn net.Conn, flag connFlag) error {
	c := &TSConn{
		fd: conn,
		transport: svr.createTransport(conn),
		flags: inboundConn,
	}

	peerhandshake, err := c.Handshake(svr.ourHandshake)

	if err != nil {
		log.Println("handshake error: ", err)
		conn.Close()
		return err
	}

	c.id = peerhandshake.ID
	c.name = peerhandshake.Name

	<- svr.pendingPeers
	svr.addPeer <- c

	return nil
}

func (svr *Server) CheckConn(peers map[NodeID]*Peer, passivePeersCount uint32) error {
	if uint32(len(peers)) >= svr.MaxPeers {
		return errors.New("to many peers.")
	}
	if passivePeersCount >= svr.MaxPassivePeers() {
		return errors.New("to many passive peers.")
	}
	return nil
}

func (svr *Server) ManageTask() {
	defer svr.waitDown.Done()

	dm := NewDialManager(svr.MaxActivePeers(), svr.BootNodes)
	var peers = make(map[NodeID]*Peer)
	var taskHasDone = make(chan Task, defaultMaxActiveDail)
	var passivePeersCount uint32 = 0
	var activeTasks []Task
	var taskQueue []Task
	delTask := func(d Task) {
		for i, t := range activeTasks {
			if t == d {
				activeTasks = append(activeTasks[:i], activeTasks[i+1:]...)
			}
		}
	}
	runTasks := func(ts []Task) []Task {
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
			newTasks := dm.CreateTasks(peers)
			taskQueue = append(taskQueue, newTasks...)
		}
	}

schedule:
	for {
		scheduleTasks()
		select {
		case <- svr.stopped:
			break schedule
		case t := <- taskHasDone:
			delTask(t)
		case c := <- svr.addPeer:
			err := svr.CheckConn(peers, passivePeersCount)
			if err == nil {
				p := NewPeer(c)
				peers[p.ID()] = p
				go svr.runPeer(p)
				passivePeersCount++
			}
		case p := <- svr.delPeer:
			delete(peers, p.ID())
			passivePeersCount--
		case fn := <- svr.peersOps:
			fn(peers)
			svr.peersOpsDone <- struct{}{}
		}
	}

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
	err := p.run()
	if err != nil {
		log.Println("peer error: ", err)
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
		log.Println(err)
		return
	}
	err = svr.SetupConn(conn, dynDialedConn)
	if err != nil {
		log.Println(err)
		return
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
		if p.ts.is(dynDialedConn) {
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
		dm.bootNodes = append(dm.bootNodes, bootNode)

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

	if len(tasks) == 0 {
		tasks = append(tasks, &waitTask{
			Duration: 3 * time.Minute,
		})
	}

	return tasks
}
func (dm *DialManager) TaskDone(t Task, time time.Time) {
	switch t2 := t.(type) {
	case *dialTask:
		delete(dm.dialing, t2.target.ID)
	case *discoverTask:
		dm.looking = false
		dm.lookResults = append(dm.lookResults, t2.results...)
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
	cpnodes := make([]*Node, 0, len((nodes)))

	var node Node
	for _, nodep := range nodes {
		node = *nodep
		cpnodes = append(cpnodes, &node)
	}

	return cpnodes
}
