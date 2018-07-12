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
	return fmt.Sprintf("%x", id)
}

func HexStr2NodeID(str string) (NodeID, error) {
	var id NodeID
	bytes, err := hex.DecodeString(strings.TrimPrefix(str, "0x"))
	if err != nil {
		return id, err
	}
	if len(bytes) != len(id) {
		return id, fmt.Errorf("unmatch length, needs %d hex chars.", len(id) * 2)
	}
	copy(id[:], bytes)
	return id, nil
}

// @section Node
type Node struct {
	ID NodeID
	IP net.IP
	Port uint16
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
	if n.IP == nil || n.IP.IsMulticast() || n.IP.IsUnspecified() {
		return errors.New("invalid ip.")
	}
	if n.Port == 0 {
		return errors.New("must has tcp port.")
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
	addr := n.IP.String() + portStr

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

func findHashFromDistance(a NodeID, d int) NodeID {
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
		return nil, errors.New("missing node id.")
	}

	id, err := HexStr2NodeID(nodeURL.User.String())
	if err != nil {
		return nil, fmt.Errorf("invalid node id. %v", err)
	}

	host, portstr, err := net.SplitHostPort(nodeURL.Host)
	if err != nil {
		return nil, fmt.Errorf("invalid host. %v", err)
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return nil, errors.New("invalid ip.")
	}

	var port uint16
	if i64, err := strconv.ParseUint(portstr, 10, 16); err != nil {
		return nil, fmt.Errorf("invalid port. ", err)
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
		nodes: make([]*Node, K),
		candidates: make([]*Node, Candidates),
	}
}

// if n exists in b.nodes, move n to head, return true.
// if b.nodes is not full, set n to the first item, return true.
// if consider n as a candidate, then unshift n to b.candidates.
// return false.
func (b *bucket) check(n *Node) bool {
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
		unshiftNode(b.nodes, n)
		b.candidates = removeNode(b.candidates, n)
		return true
	}

	return false
}
func (b *bucket) checkOrCandidate(n *Node) {
	used := b.check(n)
	if !used {
		unshiftNode(b.candidates, n)
	}
}

// obsolete the last node in b.nodes
func (b *bucket) obsolete(last *Node) *Node {
	if len(b.nodes) == 0 || b.nodes[len(b.nodes) - 1].ID != last.ID {
		return nil
	}
	if len(b.candidates) == 0 {
		b.nodes = removeNode(b.nodes, last)
		return nil
	}

	candidate := b.candidates[0]
	copy(b.candidates, b.candidates[1:])
	b.nodes[len(b.nodes) - 1] = candidate
	return candidate
}

func (b *bucket) remove(n *Node) {
	b.nodes = removeNode(b.nodes, n)
}

func removeNode(nodes []*Node, node *Node) []*Node {
	for i, n := range nodes {
		if n.ID == node.ID {
			return append(nodes[:i], nodes[i+1:]...)
		}
	}
	return nodes
}

// put node at first place of nodes without increase capacity, return the obsolete node.
func unshiftNode(nodes []*Node, node *Node) *Node {
	// if node exist in nodes, then move to first.
	var i = 0
	for i, n := range nodes {
		if n.ID == node.ID {
			nodes[0], nodes[i] = nodes[i], nodes[0]
			return nil
		}
	}

	if i < cap(nodes) {
		copy(nodes[1:], nodes)
		nodes[0] = node
		return nil
	}

	// nodes is full, obsolete the last one.
	obs := nodes[i - 1]
	for i := i - 1; i > 0; i-- {
		nodes[i] = nodes[i - 1]
	}
	nodes[0] = node
	return obs
}


// @section table

const dbCurrentVersion = 1
const seedCount = 30
const alpha = 3

var refreshDuration = 1 * time.Hour
var storeDuration = 30 * time.Minute
var checkInterval = 5 * time.Minute

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

func newTable(n *Node, net agent, dbPath string, bootNodes []*Node) (*table, error) {
	nodeDB, err := newDB(dbPath, dbCurrentVersion, n.ID)
	if err != nil {
		return nil, err
	}

	tb := &table{
		self: n,
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

	tb.loadSeedNodes()

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
	for _, node := range nodes {
		tb.addNode(node)
	}
}

func (tb *table) loop() {
	var checkTicker = time.NewTicker(checkInterval)
	var refreshTimer = time.NewTimer(refreshDuration)
	var storeTimer = time.NewTimer(storeDuration)

	var refreshDone = make(chan struct{})
	var storeDone = make(chan struct{})

	defer checkTicker.Stop()
	defer refreshTimer.Stop()
	defer storeTimer.Stop()

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
		b.obsolete(last)
	} else {
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
			return last, i
		}
	}
	return nil, 0
}

func (tb *table) refresh(done chan struct{}) {
	defer close(done)

	tb.loadSeedNodes()

	tb.lookup(tb.self.ID)

	for i := 0; i < 3; i++ {
		var target NodeID
		crand.Read(target[:])
		tb.lookup(target)
	}
}

func (tb *table) storeNodes(done chan struct{}) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	for _, b := range tb.buckets {
		for _, n := range b.nodes {
			tb.db.updateNode(n)
		}
	}
}

func (tb *table) stop() {
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
		// insert n to furtherNodeIndex
		copy(c.nodes[furtherNodeIndex + 1:], c.nodes[furtherNodeIndex:])
		c.nodes[furtherNodeIndex] = n
	}
}


// @section agent
var errStopped = errors.New("udp server has stopped.")
var errTimeout = errors.New("timeout.")

var watingTimeout = 3 * time.Minute

type DiscvConfig struct {
	Priv *ed25519.PrivateKey
	DBPath string
	BootNodes []*Node
	Addr string
}

type discover struct {
	conn net.UDPConn
	tab *table
	priv *ed25519.PrivateKey
	reqing chan *req
	getres chan *res
	stopped chan struct{}
}

func newDiscover(cfg *DiscvConfig) (*table, error) {
	addr, err := net.ResolveUDPAddr("udp", cfg.Addr)
	if err != nil {
		log.Fatal(err)
	}
	conn, err := net.ListenUDP("udp", addr)
	discv := &discover{
		conn: *conn,
		priv: cfg.Priv,
		reqing: make(chan *req),
		getres: make(chan *res),
		stopped: make(chan struct{}),
	}

	node := &Node{
		ID: priv2ID(cfg.Priv),
		IP: addr.IP,
		Port: uint16(addr.Port),
	}
	discv.tab, err = newTable(node, discv, cfg.DBPath, cfg.BootNodes)

	if err != nil {
		return nil, err
	}

	go discv.loop()
	go discv.readLoop()

	return discv.tab, nil
}

// after send query. wating for reply.
type req struct {
	senderID NodeID
	proto byte
	// if the query has been handled correctly, then return true.
	done func(Message) error
	expire time.Time
	errch chan error
}
type reqList []*req

func (rql reqList) remove(wt *req) {
	for i, w := range rql {
		if w == wt {
			copy(rql[i:], rql[i+1:])
			return
		}
	}
}

func (rql reqList) handle(rp *res) (err error) {
	var finish reqList
	for _, req := range rql {
		if req.senderID == rp.senderID && req.proto == rp.proto {
			err = req.done(rp.data)
			if err == nil {
				req.errch <- nil
				finish = append(finish, req)
			}
		}
	}
	
	// remove matched req
	for _, w := range finish {
		rql.remove(w)
	}
	
	return err
}

func (rql reqList) nextDuration() time.Duration {
	var expire time.Time

	if len(rql) == 0 {
		return 10 * time.Minute
	}

	for _, req := range rql {
		if req.expire.After(expire) {
			expire = req.expire
		}
	}

	return expire.Sub(time.Now())
}

func (rql reqList) cleanStales() {
	now := time.Now()
	var stales reqList
	for _, req := range rql {
		if req.expire.Before(now) {
			req.errch <- errTimeout
			stales = append(stales, req)
		}
	}

	for _, w := range stales {
		rql.remove(w)
	}
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
		return err
	}

	errch := d.wait(node.ID, pongCode, func(m Message) error {
		pong, ok := m.(*Pong)
		if ok {
			if pong.Ping == hash {
				return nil
			}
		}
		return errors.New("unmatched pong.")
	})

	d.conn.WriteToUDP(data, node.addr())
	return <- errch
}
func (d *discover) findnode(ID NodeID, n *Node) ([]*Node, error) {
	var nodes []*Node
	errch := d.wait(n.ID, neighborsCode, func(m Message) error {
		neighbors := m.(*Neighbors)
		nodes = neighbors.Nodes
		return nil
	})

	find := &FindNode{
		ID: ID,
		Target: ID,
	}
	d.send(n.addr(), findnodeCode, find)

	return nodes, <- errch
}

func (d *discover) wait(ID NodeID, code byte, done func(Message) error) (errch chan error) {
	errch = make(chan error, 1)
	p := &req{
		senderID: ID,
		proto: code,
		done: done,
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

	checkTimer := time.NewTimer(10 * time.Minute)
	defer checkTimer.Stop()

	for {
		select {
		case <- d.stopped:
			for _, w := range rql {
				w.errch <- errStopped
			}
		case req := <- d.reqing:
			req.expire = time.Now().Add(watingTimeout)
			rql = append(rql, req)
		case res := <- d.getres:
			err := rql.handle(res)
			res.done <- err
		case <- checkTimer.C:
			rql.cleanStales()
			checkTimer.Reset(rql.nextDuration())
		}
	}
}

func (d *discover) readLoop() {
	defer d.conn.Close()

	buf := make([]byte, maxPacketLength)
	for {
		nbytes, addr, err := d.conn.ReadFromUDP(buf)

		m, hash, err := unPacket(buf[:nbytes])
		if err != nil {
			fmt.Println("udp unpack error: ", err)
			continue
		}

		// todo
		// hash is just use for construct pong message,
		// could be optimize latter.
		m.Handle(d, addr, hash)
	}
}

func (d *discover) send(addr *net.UDPAddr, code byte, m Message) (err error) {
	data, _, err := m.Pack(d.priv)
	if err != nil {
		return err
	}

	_, err = d.conn.WriteToUDP(data, addr)
	return err
}

func (d *discover) receive(code byte, m Message) error {
	done := make(chan error, 1)
	select {
	case <- d.stopped:
		return errors.New("discover stopped.")
	case d.getres <- &res{
		senderID: *m.getID(),
		proto: code,
		data: m,
		done: done,
	}:
		return <- done
	}
}

func priv2ID(priv *ed25519.PrivateKey) (id NodeID) {
	pub := priv.PubByte()
	copy(id[:], pub)
	return
}
