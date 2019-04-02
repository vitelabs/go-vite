package discovery

import (
	"net"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

var nodePool = sync.Pool{
	New: func() interface{} {
		return &Node{
			Node: vnode.Node{
				ID:       vnode.ZERO,
				EndPoint: vnode.EndPoint{},
				Net:      0,
				Ext:      nil,
			},
			checkAt:  time.Time{},
			addAt:    time.Time{},
			activeAt: time.Time{},
			checking: false,
			finding:  false,
			addr:     nil,
		}
	},
}

func newNode() *Node {
	return nodePool.Get().(*Node)
}

func putBack(n *Node) {
	n.ID = vnode.ZERO
	n.EndPoint = vnode.EndPoint{}
	n.Net = 0
	n.Ext = nil
	n.checkAt = time.Time{}
	n.addAt = time.Time{}
	n.activeAt = time.Time{}
	n.checking = false
	n.finding = false
	n.addr = nil
	nodePool.Put(n)
}

type Node struct {
	vnode.Node
	checkAt  time.Time
	addAt    time.Time
	activeAt time.Time
	checking bool // is in check flow
	finding  bool // is finding some target from this node
	addr     *net.UDPAddr
	parseAt  time.Time // last time addr parsed
}

func (n *Node) udpAddr() (addr *net.UDPAddr, err error) {
	now := time.Now()

	if now.Sub(n.parseAt) > 15*time.Minute || n.addr == nil {
		addr, err = net.ResolveUDPAddr("udp", n.Address())
		if err != nil {
			return
		}

		n.addr = addr
		n.parseAt = now
		return
	}

	return n.addr, nil
}

// couldFind return false, if there has a find task
func (n *Node) couldFind() bool {
	return !n.finding
}

// is not checking and last check is too long ago
func (n *Node) shouldCheck() bool {
	return !n.checking && time.Now().Sub(n.checkAt) > checkExpiration
}

func (n *Node) update(n2 *Node) {
	n.ID = n2.ID
	n.Ext = n2.Ext
	n.Net = n2.Net
	n.EndPoint = n2.EndPoint
}

func extractEndPoint(addr *net.UDPAddr, from *vnode.EndPoint) (e *vnode.EndPoint, addr2 *net.UDPAddr) {
	var err error
	if from != nil {
		// from EndPoint could be unavailable
		addr2, err = net.ResolveUDPAddr("udp", from.String())
		if err != nil {
			e = udpAddrToEndPoint(addr)
			addr2 = addr
		} else {
			e = from
		}
	} else {
		e = udpAddrToEndPoint(addr)
		addr2 = addr
	}

	return
}

func udpAddrToEndPoint(addr *net.UDPAddr) (e *vnode.EndPoint) {
	e = new(vnode.EndPoint)
	if ip4 := addr.IP.To4(); len(ip4) != 0 {
		e.Host = ip4
		e.Typ = vnode.HostIPv4
	} else {
		e.Host = addr.IP
		e.Typ = vnode.HostIPv6
	}
	e.Port = addr.Port

	return
}
