package p2p

import (
	"net"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/tools/ticket"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

type dialer interface {
	dialNode(node *vnode.Node) (PeerMux, error)
}

//const maxDialCount = 3
//const retryDialDuration = 10 * time.Second
//
//type dialTask struct {
//	node  *vnode.Node
//	times int
//	err   error
//}
//
//func (t *dialTask) catch(err error) {
//	t.times++
//	t.err = err
//}

type dl struct {
	d  net.Dialer
	hk Handshaker

	cur int
	tkt ticket.Ticket

	wg sync.WaitGroup
}

func newDialer(timeout time.Duration, cur int, hk Handshaker) dialer {
	return &dl{
		d: net.Dialer{
			Timeout: timeout,
		},
		hk:  hk,
		cur: cur,
		tkt: ticket.New(cur),
	}
}

func (d *dl) dialNode(n *vnode.Node) (p PeerMux, err error) {
	conn, err := d.d.Dial("tcp", n.Address())
	if err != nil {
		return
	}

	p, err = d.hk.Handshake(conn, Inbound)

	return
}
