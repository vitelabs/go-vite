package p2p

import (
	"net"
	"fmt"
	"encoding/hex"
	"strings"
	"errors"
	"github.com/vitelabs/go-vite/p2p/protos"
	"github.com/golang/protobuf/proto"
	"math"
	"net/url"
	"strconv"
	"sync"
	mrand "math/rand"
	crand "crypto/rand"
	"encoding/binary"
	"time"
	"sort"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"log"
)

const NodeURLScheme = "vnode"

// @section NodeID
type NodeID [32]byte

func (id NodeID) String() string {
	return hex.EncodeToString(id[:])
}

func HexStr2NodeID(str string) (NodeID, error) {
	var id NodeID
	bytes, err := hex.DecodeString(strings.TrimPrefix(str, "0x"))
	if err != nil {
		return id, err
	}
	if len(bytes) != len(id) {
		return id, fmt.Errorf("unmatch length, needs %d hex chars", len(id) * 2)
	}
	copy(id[:], bytes)
	return id, nil
}

// @section Node
type Node struct {
	ID NodeID
	IP net.IP
	Port uint16
	lastping time.Time
	lastpong time.Time
}

func (n *Node) toProto() *protos.Node {
	nodepb := &protos.Node{
		ID: n.ID[:],
		IP: n.IP.String(),
		Port: uint32(n.Port),
	}
	return nodepb
}

func protoToNode(nodepb *protos.Node) *Node {
	node := new(Node)
	copy(node.ID[:], nodepb.ID)
	node.IP = net.ParseIP(nodepb.IP)
	node.Port = uint16(nodepb.Port)
	return node
}

func NewNode(id NodeID, ip net.IP, port uint16) *Node {
	return &Node{
		ID: id,
		IP: ip,
		Port: port,
	}
}

func (n *Node) Validate() error {
	if n.IP == nil {
		return errors.New("missing ip")
	}
	if n.IP.IsMulticast() || n.IP.IsUnspecified() {
		return errors.New("invalid ip")
	}
	if n.Port == 0 {
		return errors.New("missing port")
	}

	return nil
}

func (n *Node) Serialize() ([]byte, error) {
	nodepb := &protos.Node{
		IP: n.IP.String(),
		Port: uint32(n.Port),
		ID: n.ID[:],
	}

	return proto.Marshal(nodepb)
}

func (n *Node) Deserialize(bytes []byte) error {
	nodepb := &protos.Node{}
	err := proto.Unmarshal(bytes, nodepb)
	if err != nil {
		return err
	}
	n.IP = net.ParseIP(nodepb.IP)
	n.Port = uint16(nodepb.Port)
	copy(n.ID[:], nodepb.ID)
	return nil
}

func (n *Node) addr() *net.UDPAddr {
	portStr := strconv.Itoa(int(n.Port))
	addr := n.IP.String() + ":" + portStr

	udpAddr, _ := net.ResolveUDPAddr("udp", addr)

	return udpAddr
}

// @section distance

// bytes xor to distance mapping table
var matrix = [256]int{
	0, 1, 2, 2, 3, 3, 3, 3,
	4, 4, 4, 4, 4, 4, 4, 4,
	5, 5, 5, 5, 5, 5, 5, 5,
	5, 5, 5, 5, 5, 5, 5, 5,
	6, 6, 6, 6, 6, 6, 6, 6,
	6, 6, 6, 6, 6, 6, 6, 6,
	6, 6, 6, 6, 6, 6, 6, 6,
	6, 6, 6, 6, 6, 6, 6, 6,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
	8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8,
	8, 8, 8, 8, 8, 8, 8, 8,
}

// xor every byte of a and b from left to right.
// stop at the first different byte (for brevity, we call it FDB).
// distance of a and b is bits-count of the FDB plus the bits-count of rest bytes.
func calcDistance(a, b NodeID) int {
	delta := 0
	var i int
	for i := range a {
		x := a[i] ^ b[i]
		if x != 0 {
			delta += matrix[x]
			break
		}
	}

	return delta + (len(a) - i - 1) * 8
}

func disCmp(target, a, b NodeID) int {
	var cmp byte
	for i := range target {
		cmp = a[i] ^ target[i] - b[i] ^ target[i]
		if cmp > 0 {
			return 1
		}
		if cmp < 0 {
			return -1
		}
	}

	return 0
}

func findNodeIDFromDistance(a NodeID, d int) NodeID {
	if d == 0 {
		return a
	}
	b := a

	// pos mean the FDB between a and b from left to right.
	pos := len(a) - d / 8 - 1

	var xor byte = byte(d % 8)
	// mean the xor of FDB is greater or equal 127.
	if xor == 0 {
		pos++
		xor = byte(randInt(127, 256))
	} else {
		xor = expRand(xor)
	}
	// if byte1 xor byte2 get d,
	// then byte2 can be calc from (byte1^d | ^byte1&d)
	b[pos] = a[pos]&^xor | ^a[pos]&xor

	// fill the rest bytes.
	for i := pos + 1; i < len(a); i++ {
		b[i] = byte(mrand.Intn(255))
	}

	return b
}

func randInt(min, max int) int {
	return mrand.Intn(max - min) + min
}

// get rand int in [2**(n-1), 2**n)
func expRand(n byte) byte {
	low, up := int(math.Pow(2.0, float64(n - 1))), int(math.Pow(2.0, float64(n)))
	return byte(randInt(low, up))
}

// @section NodeURL

// marshal node to url-like string which looks like:
// vnode://<hex node id>@<ip>:<port>
func (n *Node) String() string {
	nodeURL := url.URL{
		Scheme: NodeURLScheme,
	}

	err := n.Validate()
	if err == nil {
		addr := net.TCPAddr{
			IP: n.IP,
			Port: int(n.Port),
		}
		nodeURL.User = url.User(n.ID.String())
		nodeURL.Host = addr.String()
	} else {
		nodeURL.Host = n.ID.String()
	}
	return nodeURL.String()
}

// parse a url-like string to Node
func ParseNode(u string) (*Node, error) {
	nodeURL, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	if nodeURL.Scheme != NodeURLScheme {
		return nil, errors.New(`invalid scheme, should be "vnode"`)
	}
	if nodeURL.User == nil {
		return nil, errors.New("missing NodeID")
	}

	id, err := HexStr2NodeID(nodeURL.User.String())
	if err != nil {
		return nil, fmt.Errorf("invalid node id: %v", err)
	}

	host, portstr, err := net.SplitHostPort(nodeURL.Host)
	if err != nil {
		return nil, fmt.Errorf("invalid host: %v", err)
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return nil, errors.New("invalid ip")
	}

	var port uint16
	if i64, err := strconv.ParseUint(portstr, 10, 16); err != nil {
		return nil, fmt.Errorf("invalid port: %v\n", err)
	} else {
		port = uint16(i64)
	}

	return NewNode(id, ip, port), nil
}


// @section Bucket

const K = 16
const N = 17
const Candidates = 10
const minDistance = 239

type bucket struct {
	nodes []*Node
	candidates []*Node
}

func NewBucket() *bucket {
	return &bucket{
		nodes: make([]*Node, 0, K),
		candidates: make([]*Node, 0, Candidates),
	}
}

// if n exists in b.nodes, move n to head, return true.
// if b.nodes is not full, set n to the first item, return true.
// if consider n as a candidate, then unshift n to b.candidates.
// return false.
func (b *bucket) check(n *Node) bool {
	if n == nil {
		return false
	}

	var i = 0
	for i, node := range b.nodes {
		if node == nil {
			break
		}
		// if n exists in b, then move n to head.
		if node.ID == n.ID {
			for j := i; j > 0; j-- {
				b.nodes[j] = b.nodes[j-1]
			}
			b.nodes[0] = n
			return true
		}
	}

	// if nodes is not full, set n to the first item
	if i < cap(b.nodes) {
		b.nodes = unshiftNode(b.nodes, n)
		b.candidates = removeNode(b.candidates, n)
		return true
	}

	return false
}
func (b *bucket) checkOrCandidate(n *Node) {
	used := b.check(n)
	if !used {
		b.candidates = unshiftNode(b.candidates, n)
	}
}

// obsolete the last node in b.nodes and return replacer.
func (b *bucket) obsolete(last *Node) *Node {
	if len(b.nodes) == 0 || b.nodes[len(b.nodes) - 1].ID != last.ID {
		return nil
	}
	if len(b.candidates) == 0 {
		b.nodes = b.nodes[:len(b.nodes)-1]
		return nil
	}

	candidate := b.candidates[0]
	b.candidates = b.candidates[1:]
	b.nodes[len(b.nodes) - 1] = candidate
	return candidate
}

func (b *bucket) remove(n *Node) {
	b.nodes = removeNode(b.nodes, n)
}

func removeNode(nodes []*Node, node *Node) []*Node {
	if node == nil {
		return nodes
	}

	for i, n := range nodes {
		if n != nil && n.ID == node.ID {
			return append(nodes[:i], nodes[i+1:]...)
		}
	}
	return nodes
}

// put node at first place of nodes without increase capacity, return the obsolete node.
func unshiftNode(nodes []*Node, node *Node) []*Node {
	if node == nil {
		return nil
	}

	// if node exist in nodes, then move to first.
	i := 0
	for _, n := range nodes {
		i++
		if n != nil && n.ID == node.ID {
			nodes[0], nodes[i] = nodes[i], nodes[0]
			return nodes
		}
	}

	// i equals len(nodes)
	if i < cap(nodes) {
		return append([]*Node{node}, nodes...)
	}

	// nodes is full, obsolete the last one.
	for i = i - 1; i > 0; i-- {
		nodes[i] = nodes[i - 1]
	}
	nodes[0] = node
	return nodes
}


// @section table

const dbCurrentVersion = 1
const seedCount = 30
const alpha = 3

var refreshDuration = 30 * time.Minute
var storeDuration = 5 * time.Minute
var checkInterval = 3 * time.Minute
var watingTimeout = 2 * time.Minute	// watingTimeout must be enough little, at least than checkInterval
var pingInvervalPerNode = 10 * time.Minute

type table struct {
	mutex   sync.Mutex
	buckets [N]*bucket
	bootNodes []*Node
	db         *nodeDB
	nodeAddedHook func(*Node)
	agent agent
	self *Node
	rand *mrand.Rand
	stopped chan struct{}
}

type agent interface {
	ping(*Node) error
	findnode(NodeID, *Node) ([]*Node, error)
	close()
}

func newTable(self *Node, net agent, dbPath string, bootNodes []*Node) (*table, error) {
	nodeDB, err := newDB(dbPath, dbCurrentVersion, self.ID)
	if err != nil {
		return nil, err
	}

	tb := &table{
		self: self,
		db: nodeDB,
		agent: net,
		bootNodes: bootNodes,
		rand: mrand.New(mrand.NewSource(0)),
		stopped: make(chan struct{}),
	}

	// init buckets
	for i, _ := range tb.buckets {
		tb.buckets[i] = NewBucket()
	}

	tb.resetRand()

	go tb.loop()

	return tb, nil
}

func (tb *table) resetRand() {
	var b [8]byte
	crand.Read(b[:])

	tb.mutex.Lock()
	tb.rand.Seed(int64(binary.BigEndian.Uint64(b[:])))
	tb.mutex.Unlock()
}

func (tb *table) loadSeedNodes() {
	nodes := tb.db.randomNodes(seedCount)
	nodes = append(nodes, tb.bootNodes...)
	log.Printf("discover table load %d seed nodes\n", len(nodes))

	for _, node := range nodes {
		tb.addNode(node)
	}
}

func (tb *table) loop() {
	checkTicker := time.NewTicker(checkInterval)
	refreshTimer := time.NewTimer(refreshDuration)
	storeTimer := time.NewTimer(storeDuration)

	defer checkTicker.Stop()
	defer refreshTimer.Stop()
	defer storeTimer.Stop()

	refreshDone := make(chan struct{})
	storeDone := make(chan struct{})

	// initial refresh
	go tb.refresh(refreshDone)

loop:
	for {
		select {
		case <- refreshTimer.C:
			go tb.refresh(refreshDone)
		case <- refreshDone:
			refreshTimer.Reset(refreshDuration)

		case <-checkTicker.C:
			go tb.checkLastNode()

		case <-storeTimer.C:
			go tb.storeNodes(storeDone)
		case <- storeDone:
			storeTimer.Reset(storeDuration)
		case <-tb.stopped:
			break loop
		}
	}

	if tb.agent != nil {
		tb.agent.close()
	}

	tb.db.close()
}

func (tb *table) addNode(node *Node) {
	if node == nil {
		return
	}
	if node.ID == tb.self.ID {
		return
	}

	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	bucket := tb.getBucket(node.ID)
	bucket.checkOrCandidate(node)
}
func (tb *table) getBucket(id NodeID) *bucket {
	d := calcDistance(tb.self.ID, id)
	if d <= minDistance {
		return tb.buckets[0]
	}
	return tb.buckets[d-minDistance-1]
}

func (tb *table) delete(node *Node) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	bucket := tb.getBucket(node.ID)
	bucket.remove(node)
}

func (tb *table) checkLastNode() {
	last, bi := tb.pickLastNode()
	if last == nil {
		return
	}

	err := tb.agent.ping(last)

	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	b := tb.buckets[bi]

	if err != nil {
		log.Printf("obsolete %s: %v\n", last.ID, err)
		b.obsolete(last)
	} else {
		log.Printf("check %s\n", last.ID)
		b.check(last)
	}
}

func (tb *table) pickLastNode() (*Node, int) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	for _, i := range tb.rand.Perm(len(tb.buckets)) {
		b := tb.buckets[i]
		if len(b.nodes) > 0 {
			last := b.nodes[len(b.nodes) - 1]
			if time.Now().Sub(last.lastping) > pingInvervalPerNode {
				return last, i
			}
		}
	}
	return nil, 0
}

func (tb *table) refresh(done chan struct{}) {
	log.Println("discv table begin refresh")

	tb.loadSeedNodes()

	tb.lookup(tb.self.ID)

	for i := 0; i < 3; i++ {
		var target NodeID
		crand.Read(target[:])
		tb.lookup(target)
	}

	done <- struct{}{}
}

func (tb *table) storeNodes(done chan struct{}) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	for _, b := range tb.buckets {
		for _, n := range b.nodes {
			tb.db.updateNode(n)
		}
	}

	done <- struct{}{}
}

func (tb *table) stop() {
	log.Println("discv table stop")
	close(tb.stopped)
}

func (tb *table) lookup(target NodeID) []*Node {
	var asked = make(map[NodeID]bool)
	var seen = make(map[NodeID]bool)
	var reply = make(chan []*Node, alpha)
	var queries = 0
	var result *closest

	asked[tb.self.ID] = true

	for {
		tb.mutex.Lock()
		result = tb.closest(target, K)
		tb.mutex.Unlock()
		if len(result.nodes) > 0 {
			break
		}
	}

	for {
		for i := 0; i < len(result.nodes) && queries < alpha; i++ {
			n := result.nodes[i]
			if !asked[n.ID] {
				asked[n.ID] = true
				queries++
				go tb.findnode(n, target, reply)
			}
		}
		if queries == 0 {
			break
		}

		for _, n := range <-reply {
			if n != nil && !seen[n.ID] {
				seen[n.ID] = true
				result.push(n, K)
			}
		}
		queries--
	}
	return result.nodes
}

func (tb *table) closest(target NodeID, count int) *closest {
	result := &closest{target: target}
	for _, b := range tb.buckets {
		for _, n := range b.nodes {
			result.push(n, count)
		}
	}
	return result
}

func (tb *table) findnode(n *Node, targetID NodeID, reply chan<- []*Node) {
	nodes, err := tb.agent.findnode(targetID, n)

	if err != nil || len(nodes) == 0 {
		tb.delete(n)
	}

	for _, n := range nodes {
		tb.addNode(n)
	}

	reply <- nodes
}


// @section closet
// closest nodes to the target NodeID
type closest struct {
	nodes []*Node
	target NodeID
}

func (c *closest) push(n *Node, count int)  {
	if n == nil {
		return
	}

	length := len(c.nodes)
	furtherNodeIndex := sort.Search(length, func(i int) bool {
		return disCmp(c.target, c.nodes[i].ID, n.ID) > 0
	})

	// closest Nodes list is full.
	if length >= count {
		// replace the further one.
		if furtherNodeIndex < length {
			c.nodes[furtherNodeIndex] = n
		}
	} else {
		// increase c.nodes length first.
		c.nodes = append(c.nodes, nil)
		// insert n to furtherNodeIndex
		copy(c.nodes[furtherNodeIndex + 1:], c.nodes[furtherNodeIndex:])
		c.nodes[furtherNodeIndex] = n
	}
}


// @section agent
var errStopped = errors.New("discv server has stopped")
var errTimeout = errors.New("timeout")

type DiscvConfig struct {
	Priv ed25519.PrivateKey
	DBPath string
	BootNodes []*Node
	Addr *net.UDPAddr
}

type discover struct {
	conn *net.UDPConn
	tab *table
	priv ed25519.PrivateKey
	reqing chan *req
	getres chan *res
	stopped chan struct{}
}

func newDiscover(cfg *DiscvConfig) (*table, *net.UDPAddr, error) {
	laddr := cfg.Addr
	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		log.Fatalf("discv listen udp error: %v\n", err)
	}
	discv := &discover{
		conn: conn,
		priv: cfg.Priv,
		reqing: make(chan *req),
		getres: make(chan *res),
		stopped: make(chan struct{}),
	}

	// get the real local address. eg. 127.0.0.1:8483
	laddr = conn.LocalAddr().(*net.UDPAddr)

	// get the publicIP announced to other nodes.
	var publicIP net.IP
	extIP, err := getExtIP()
	if err != nil {
		log.Printf("got external ip error: %v\n", err)
		publicIP = laddr.IP
	} else {
		publicIP = extIP
	}

	node := &Node{
		ID: priv2ID(cfg.Priv),
		IP: publicIP,
		Port: uint16(laddr.Port),
	}
	fmt.Println("self node: ", *node)
	log.Printf("self: %s\n", node)

	discv.tab, err = newTable(node, discv, cfg.DBPath, cfg.BootNodes)

	if err != nil {
		return nil, nil, err
	}

	go discv.loop()
	go discv.readLoop()

	return discv.tab, laddr, nil
}

// after send query. wating for reply.
type req struct {
	senderID NodeID
	proto byte
	// if the query has been handled correctly, then return true.
	callback func(Message) error
	expire time.Time
	errch chan error
}
type reqList []*req

func handleRes(rql reqList, rp *res) (rest reqList) {
	var err error
	for i, req := range rql {
		if req.senderID == rp.senderID && req.proto == rp.proto {
			if req.callback != nil {
				err = req.callback(rp.data)
			}

			rp.done <- err
			req.errch <- err

			rest = rql[:i]
			rest = append(rest, rql[i+1:]...)
			return rql
		}
	}

	return rql
}

func cleanStaleReq(rql reqList) reqList {
	now := time.Now()
	rest := make(reqList, 0, len(rql))
	for _, req := range rql {
		if req.expire.Before(now) {
			req.errch <- errTimeout
		} else {
			rest = append(rest, req)
		}
	}
	return rest
}

type res struct {
	senderID NodeID
	proto byte
	data Message
	done chan error
}

func (d *discover) getID() NodeID {
	return d.tab.self.ID
}

// implements table.agent interface
func (d *discover) ping(node *Node) error {
	ping := &Ping{
		ID: d.getID(),
	}
	data, hash, err := ping.Pack(d.priv)
	if err != nil {
		return fmt.Errorf("pack discv ping msg error: %v\n", err)
	}

	n, err := d.conn.WriteToUDP(data, node.addr())
	if err != nil {
		return fmt.Errorf("send ping to %s error: %v\n", node, err)
	}
	if n != len(data) {
		return fmt.Errorf("send incomplete ping to %s: %d/%d\n", node, n, len(data))
	}
	log.Printf("send ping to %s\n", node)
	node.lastping = time.Now()

	errch := d.wait(node.ID, pongCode, func(m Message) error {
		pong, ok := m.(*Pong)
		if ok {
			if pong.Ping == hash {
				node.lastping = time.Now()
				return nil
			}
		}
		return errors.New("unmatched pong")
	})

	return <- errch
}
func (d *discover) findnode(ID NodeID, n *Node) (nodes []*Node, err error) {
	log.Printf("findnode %s to %s\n", ID, n)

	err = d.send(n.addr(), findnodeCode, &FindNode{
		ID: d.getID(),
		Target: ID,
	})
	if err != nil {
		return nodes, err
	}

	errch := d.wait(n.ID, neighborsCode, func(m Message) error {
		neighbors, ok := m.(*Neighbors)
		if !ok {
			return fmt.Errorf("receive unmatched msg, should be neighbors\n")
		}
		nodes = neighbors.Nodes
		return nil
	})

	err = <- errch
	log.Printf("findnode got %d nodes, error: %v\n", len(nodes), err)
	return nodes, err
}

func (d *discover) wait(ID NodeID, code byte, callback func(Message) error) (errch chan error) {
	errch = make(chan error, 1)
	p := &req{
		senderID: ID,
		proto: code,
		callback: callback,
		expire: time.Now().Add(watingTimeout),
		errch: errch,
	}
	select {
	case d.reqing <- p:
	case <- d.stopped:
		errch <- errStopped
	}

	return errch
}

func (d *discover) close() {
	close(d.stopped)
	d.conn.Close()
}

func (d *discover) loop() {
	var rql reqList

	checkTicker := time.NewTicker(watingTimeout / 2)
	defer checkTicker.Stop()

	for {
		select {
		case <- d.stopped:
			for _, w := range rql {
				w.errch <- errStopped
			}
		case req := <- d.reqing:
			log.Printf("wating msg %d from %s expire %s\n", req.proto, req.senderID, req.expire)
			rql = append(rql, req)
		case res := <- d.getres:
			rql = handleRes(rql, res)
			log.Printf("handle msg %d from %s\n", res.proto, res.senderID)
		case <- checkTicker.C:
			rql = cleanStaleReq(rql)
		}
	}
}

func (d *discover) readLoop() {
	defer d.conn.Close()

	buf := make([]byte, maxPacketLength)
	for {
		nbytes, addr, err := d.conn.ReadFromUDP(buf)

		if nbytes == 0 {
			log.Printf("discv read from %s 0 bytes\n", addr)
			continue
		}

		m, hash, err := unPacket(buf[:nbytes])
		if err != nil {
			log.Printf("udp unpack from %s error: %v\n", addr, err)
			continue
		}

		log.Printf("udp read from %s@%s\n", m.getID(), addr)

		// todo
		// hash is just use for construct pong message,
		// could be optimize latter.
		err = m.Handle(d, addr, hash)
		if err != nil {
			log.Printf("handle discv msg from %s@%s error: %v\n", m.getID(), addr, err)
		}
	}
}

func (d *discover) send(addr *net.UDPAddr, code byte, m Message) (err error) {
	data, _, err := m.Pack(d.priv)
	if err != nil {
		return err
	}

	n, err := d.conn.WriteToUDP(data, addr)
	if err != nil {
		return fmt.Errorf("send msg %d to %s error: %v\n", code, addr, err)
	}
	if n != len(data) {
		return fmt.Errorf("send incomplete msg to %s: %d/%d\n", addr, n, len(data))
	}

	log.Printf("send msg %d (%d bytes) to %s\n", code, n, addr)

	return nil
}

func (d *discover) receive(code byte, m Message) error {
	done := make(chan error, 1)
	select {
	case <- d.stopped:
		return errors.New("discover stopped")
	case d.getres <- &res{
		senderID: m.getID(),
		proto: code,
		data: m,
		done: done,
	}:
		return <- done
	}
}

func priv2ID(priv ed25519.PrivateKey) (id NodeID) {
	pub := priv.PubByte()
	copy(id[:], pub)
	return
}
