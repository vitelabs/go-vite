package discovery

import (
	"net"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/net/netool"
	"github.com/vitelabs/go-vite/net/vnode"
)

type Node struct {
	vnode.Node
	checkAt  int64
	addAt    int64
	activeAt int64

	finding int32
	findAt  int64

	addr    *net.UDPAddr
	parseAt int64 // last time addr parsed
}

func (n *Node) udpAddr() (addr *net.UDPAddr, err error) {
	now := time.Now().Unix()

	// 15min
	if now-n.parseAt > 900 || n.addr == nil {
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

func (n *Node) update(n2 *Node) {
	n.ID = n2.ID
	n.Ext = n2.Ext
	n.Net = n2.Net
	n.EndPoint = n2.EndPoint
}

func (n *Node) needCheck() bool {
	// 10 minutes
	now := time.Now().Unix()
	if now-n.checkAt > 60*10 {
		return true
	}

	return false
}

func (n *Node) couldFind() bool {
	now := time.Now().Unix()
	// 1 minute
	if now-n.findAt < 60 {
		return false
	}

	if !atomic.CompareAndSwapInt32(&n.finding, 0, 1) {
		return false
	}

	return true
}

func (n *Node) findDone() {
	atomic.StoreInt32(&n.finding, 0)
	n.findAt = time.Now().Unix()
}

func extractEndPoint(addr *net.UDPAddr, from *vnode.EndPoint) (e *vnode.EndPoint, addr2 *net.UDPAddr) {
	var err error
	var done bool
	if from != nil {
		// from is available
		addr2, err = net.ResolveUDPAddr("udp", from.String())
		if err == nil {
			if from.Typ.Is(vnode.HostDomain) || netool.CheckRelayIP(addr.IP, from.Host) == nil {
				// from is domain, or IP is available
				done = true
				e = from
			}
		}
	}

	if !done {
		e = udpAddrToEndPoint(addr)
		addr2 = addr
	}

	return
}

func nodeFromEndPoint(e *vnode.EndPoint) (n *Node, err error) {
	udp, err := net.ResolveUDPAddr("udp", e.String())
	if err != nil {
		return
	}

	return &Node{
		Node: vnode.Node{
			EndPoint: *e,
		},
		addr:    udp,
		parseAt: time.Now().Unix(),
	}, nil
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

func nodeFromPing(res *packet) *Node {
	p := res.body.(*ping)

	e, addr := extractEndPoint(res.from, p.from)

	return &Node{
		Node: vnode.Node{
			ID:       res.id,
			EndPoint: *e,
			Net:      p.net,
			Ext:      p.ext,
		},
		addr:    addr,
		parseAt: time.Now().Unix(),
	}
}

func nodeFromPong(res *packet) *Node {
	p := res.body.(*pong)

	e, addr := extractEndPoint(res.from, p.from)

	return &Node{
		Node: vnode.Node{
			ID:       res.id,
			EndPoint: *e,
			Net:      p.net,
			Ext:      p.ext,
		},
		addr:    addr,
		parseAt: time.Now().Unix(),
	}
}
