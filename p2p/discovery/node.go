package discovery

import (
	"encoding/hex"
	"errors"
	"math"
	mrand "math/rand"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/p2p/discovery/protos"
	"github.com/vitelabs/go-vite/p2p/network"
)

// @section NodeID
var errUnmatchedLength = errors.New("unmatch length, needs 64 hex chars")

// ZERO_NODE_ID is the zero-value of NodeID type
var ZERO_NODE_ID NodeID

// NodeID use to mark node, and build a structural network
type NodeID [32]byte

// Bytes return a slice derived from NodeID
func (id NodeID) Bytes() []byte {
	return id[:]
}

// String return a hex coded string of NodeID
func (id NodeID) String() string {
	return hex.EncodeToString(id[:])
}

// Brief return the front 4 bytes hex coded string of NodeID
func (id NodeID) Brief() string {
	return hex.EncodeToString(id[:4])
}

// IsZero validate whether a NodeID is zero-value
func (id NodeID) IsZero() bool {
	for _, byt := range id {
		if byt|0 != 0 {
			return false
		}
	}
	return true
}

// Equal validate whether two NodeID is equal
func (id NodeID) Equal(id2 NodeID) bool {
	for i := 0; i < 32; i++ {
		if id[i]^id2[i] != 0 {
			return false
		}
	}

	return true
}

// HexStr2NodeID parse a hex coded string to NodeID
func HexStr2NodeID(str string) (id NodeID, err error) {
	bytes, err := hex.DecodeString(strings.TrimPrefix(str, "0x"))
	if err != nil {
		return
	}

	return Bytes2NodeID(bytes)
}

// Bytes2NodeID turn a slice to NodeID
func Bytes2NodeID(buf []byte) (id NodeID, err error) {
	if len(buf) != len(id) {
		return id, errUnmatchedLength
	}

	copy(id[:], buf)
	return
}

// @section Node
var errMissID = errors.New("missing NodeID")
var errMissIP = errors.New("missing IP")
var errInvalidIP = errors.New("invalid IP")
var errMissPort = errors.New("missing port")
var errInvalidScheme = errors.New("invalid scheme")

// Node mean a node in vite P2P network
type Node struct {
	ID      NodeID
	IP      net.IP
	UDP     uint16
	TCP     uint16
	Net     network.ID
	Ext     []byte
	Version byte

	addAt    time.Time
	lastPing time.Time
	activeAt time.Time
	weight   int64 // tcp connection lifetime, longer is better
	findfail int
}

func (n *Node) proto() *protos.Node {
	return &protos.Node{
		ID:  n.ID[:],
		IP:  n.IP,
		UDP: uint32(n.UDP),
		TCP: uint32(n.TCP),
		Net: uint64(n.Net),
		Ext: n.Ext,
	}
}

func protoToNode(pb *protos.Node) (*Node, error) {
	id, err := Bytes2NodeID(pb.ID)
	if err != nil {
		return nil, err
	}

	node := new(Node)
	node.ID = id
	node.IP = pb.IP
	node.UDP = uint16(pb.UDP)
	node.TCP = uint16(pb.TCP)
	node.Net = network.ID(pb.Net)
	node.Ext = pb.Ext

	return node, nil
}

// Validate whether a node has essential information
func (n *Node) Validate() error {
	if n.ID.IsZero() {
		return errMissID
	}

	if n.IP == nil {
		return errMissIP
	}

	//if n.IP.IsLoopback() || n.IP.IsMulticast() || n.IP.IsUnspecified() {
	//	return errInvalidIP
	//}

	if n.UDP == 0 {
		return errMissPort
	}

	return nil
}

// Serialize a Node to []byte
func (n *Node) Serialize() ([]byte, error) {
	return proto.Marshal(n.proto())
}

// Deserialize encoded data, []byte, to a Node,
// you must create the Node first, like following:
//		n := new(Node)
//		err := n.Deserialize(buf)
func (n *Node) Deserialize(bytes []byte) error {
	pb := new(protos.Node)
	err := proto.Unmarshal(bytes, pb)
	if err != nil {
		return err
	}

	n2, err := protoToNode(pb)
	if err != nil {
		return err
	}

	*n = *n2

	return nil
}

// UDPAddr return the address that can communication with udp
func (n *Node) UDPAddr() *net.UDPAddr {
	return &net.UDPAddr{
		IP:   n.IP,
		Port: int(n.UDP),
	}
}

// TCPAddr return the address that can be connected with tcp
func (n *Node) TCPAddr() *net.TCPAddr {
	port := n.TCP
	if port == 0 {
		port = n.UDP
	}
	return &net.TCPAddr{
		IP:   n.IP,
		Port: int(port),
	}
}

// @section NodeURL
const NodeURLScheme = "vnode"

// String marshal node to url-like string which looks like:
// 	vnode://<hex node id>
// 	vnode://<hex node id>@<ip>:<udpPort>#<tcpPort>
func (n *Node) String() string {
	nodeURL := url.URL{
		Scheme: NodeURLScheme,
	}

	err := n.Validate()
	if err == nil {
		nodeURL.User = url.User(n.ID.String())
		nodeURL.Host = n.UDPAddr().String()
		if n.TCP != 0 && n.TCP != n.UDP {
			nodeURL.Fragment = strconv.Itoa(int(n.TCP))
		}
	} else {
		nodeURL.Host = n.ID.String()
	}

	nodeURL.RawQuery = "netid=" + strconv.FormatUint(uint64(n.Net), 10)

	return nodeURL.String()
}

// ParseNode parse a url-like string to Node
func ParseNode(u string) (*Node, error) {
	nodeURL, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	if nodeURL.Scheme != NodeURLScheme {
		return nil, errInvalidScheme
	}
	if nodeURL.User == nil {
		return nil, errMissID
	}

	id, err := HexStr2NodeID(nodeURL.User.String())
	if err != nil {
		return nil, err
	}

	host, port, err := net.SplitHostPort(nodeURL.Host)
	if err != nil {
		return nil, err
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return nil, errInvalidIP
	}

	var udp, tcp uint16
	udp, err = parsePort(port)
	if err != nil {
		return nil, err
	}

	if nodeURL.Fragment != "" {
		tcp, err = parsePort(nodeURL.Fragment)
		if err != nil {
			return nil, err
		}
	} else {
		tcp = udp
	}

	var netid uint64
	query := nodeURL.Query()
	if query.Get("netid") != "" {
		var nid uint64
		if nid, err = strconv.ParseUint(query.Get("netid"), 10, 64); err == nil {
			netid = nid
		}
	}

	return &Node{
		ID:  id,
		IP:  ip,
		UDP: udp,
		TCP: tcp,
		Net: network.ID(netid),
	}, nil
}

func parsePort(str string) (port uint16, err error) {
	i, err := strconv.ParseUint(str, 10, 16)
	if err != nil {
		return
	}

	return uint16(i), nil
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
// term at the first different byte (for brevity, we call it FDB).
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

	return delta + (len(a)-i-1)*8
}

// (distance between target and a) compare to (distance between target and b)
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
	pos := len(a) - d/8 - 1

	xor := byte(d % 8)
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
	return mrand.Intn(max-min) + min
}

// get rand int in [2**(n-1), 2**n)
func expRand(n byte) byte {
	low, up := int(math.Pow(2.0, float64(n-1))), int(math.Pow(2.0, float64(n)))
	return byte(randInt(low, up))
}
