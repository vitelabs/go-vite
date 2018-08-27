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

const NodeURLScheme = "vnode"

// @section NodeID
type NodeID [32]byte

func (id NodeID) String() string {
	return hex.EncodeToString(id[:])
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
type EndPoint struct {
	IP   net.IP
	Port uint16
}

func (e *EndPoint) Addr() string {
	return e.IP.String() + ":" + strconv.Itoa(int(e.Port))
}

func (e *EndPoint) String() string {
	return e.Addr()
}

type Node struct {
	ID       NodeID
	IP       net.IP
	Port     uint16
	lastping time.Time
	lastpong time.Time
}

func (n *Node) toProto() *protos.Node {
	return &protos.Node{
		ID:   n.ID[:],
		IP:   n.IP.String(),
		Port: uint32(n.Port),
	}
}

func protoToNode(nodepb *protos.Node) *Node {
	node := new(Node)
	copy(node.ID[:], nodepb.ID)
	node.IP = net.ParseIP(nodepb.IP)
	node.Port = uint16(nodepb.Port)
	return node
}

func newNode(id NodeID, ip net.IP, port uint16) *Node {
	return &Node{
		ID:   id,
		IP:   ip,
		Port: port,
	}
}

func (n *Node) EndPoint() *EndPoint {
	return &EndPoint{
		IP:   n.IP,
		Port: n.Port,
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
		IP:   n.IP.String(),
		Port: uint32(n.Port),
		ID:   n.ID[:],
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
	return &net.UDPAddr{
		IP:   n.IP,
		Port: int(n.Port),
	}
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
			IP:   n.IP,
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

	return &Node{
		ID:   id,
		IP:   ip,
		Port: port,
	}, nil
}
