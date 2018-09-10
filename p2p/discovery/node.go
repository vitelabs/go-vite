package discovery

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/p2p/discovery/protos"
	"math"
	mrand "math/rand"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// @section NodeID
type NodeID [32]byte

func (id NodeID) String() string {
	return hex.EncodeToString(id[:])
}

func (id NodeID) Brief() string {
	return hex.EncodeToString(id[:4])
}

func (id NodeID) IsZero() bool {
	for _, byt := range id {
		if byt != 0 {
			return false
		}
	}
	return true
}

func HexStr2NodeID(str string) (id NodeID, err error) {
	bytes, err := hex.DecodeString(strings.TrimPrefix(str, "0x"))
	if err != nil {
		return id, err
	}
	if len(bytes) != len(id) {
		return id, fmt.Errorf("unmatch length, needs %d hex chars", len(id)*2)
	}
	copy(id[:], bytes)
	return id, nil
}

// @section Node
//type EndPoint struct {
//	IP   net.IP
//	Port uint16
//}
//
//func (e *EndPoint) Addr() string {
//	return e.IP.String() + ":" + strconv.Itoa(int(e.Port))
//}
//
//func (e *EndPoint) String() string {
//	return e.Addr()
//}

var errNullID = errors.New("missing ID")
var errMissIP = errors.New("missing IP")
var errInvalidIP = errors.New("invalid IP")
var errMissPort = errors.New("missing port")

type Node struct {
	ID               NodeID
	IP               net.IP
	UDP              uint16
	TCP              uint16
	lastPing         time.Time
	lastPong         time.Time
	lastPingReceived time.Time
	lastPongReceived time.Time
}

func (n *Node) toProto() *protos.Node {
	return &protos.Node{
		ID:  n.ID[:],
		IP:  n.IP,
		UDP: uint32(n.UDP),
		TCP: uint32(n.TCP),
	}
}

func protoToNode(nodepb *protos.Node) *Node {
	node := new(Node)
	copy(node.ID[:], nodepb.ID)
	node.IP = nodepb.IP
	node.UDP = uint16(nodepb.UDP)
	node.TCP = uint16(nodepb.TCP)

	return node
}

//func (n *Node) EndPoint() *EndPoint {
//	return &EndPoint{
//		IP:   n.IP,
//		Port: n.Port,
//	}
//}

func (n *Node) Validate() error {
	if n.ID.IsZero() {
		return errNullID
	}

	if n.IP == nil {
		return errMissIP
	}

	if n.IP.IsLoopback() || n.IP.IsMulticast() || n.IP.IsUnspecified() {
		return errInvalidIP
	}

	if n.UDP == 0 {
		return errMissPort
	}

	return nil
}

func (n *Node) Serialize() ([]byte, error) {
	return proto.Marshal(n.toProto())
}

func (n *Node) Deserialize(bytes []byte) error {
	nodepb := new(protos.Node)
	err := proto.Unmarshal(bytes, nodepb)
	if err != nil {
		return err
	}

	n.IP = nodepb.IP
	n.UDP = uint16(nodepb.UDP)
	n.TCP = uint16(nodepb.TCP)
	copy(n.ID[:], nodepb.ID)

	return nil
}

func (n *Node) UDPAddr() *net.UDPAddr {
	return &net.UDPAddr{
		IP:   n.IP,
		Port: int(n.UDP),
	}
}
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

// marshal node to url-like string which looks like:
// vnode://<hex node id>
// vnode://<hex node id>@<ip>:<udpPort>#<tcpPort>
func (n *Node) String() string {
	nodeURL := url.URL{
		Scheme: NodeURLScheme,
	}

	err := n.Validate()
	if err == nil {
		addr := n.UDPAddr()
		nodeURL.User = url.User(n.ID.String())
		nodeURL.Host = addr.String()
		if n.TCP != 0 && n.TCP != n.UDP {
			nodeURL.Fragment = strconv.Itoa(int(n.TCP))
		}
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
		return nil, fmt.Errorf("invalid scheme, should be %s\n", NodeURLScheme)
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

	var udp, tcp uint16
	udp, err = parsePort(portstr)
	if err != nil {
		return nil, fmt.Errorf("invalid udp port: %v\n", err)
	}

	if nodeURL.Fragment != "" {
		tcp, err = parsePort(portstr)
		if err != nil {
			return nil, fmt.Errorf("invalid tcp port: %v\n", err)
		}
	} else {
		tcp = udp
	}

	return &Node{
		ID:  id,
		IP:  ip,
		UDP: udp,
		TCP: tcp,
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
