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
	net.Dialer
	handshaker   Handshaker
	codecFactory CodecFactory
	cur          int
	tkt          ticket.Ticket
	wg           sync.WaitGroup
}

func newDialer(timeout time.Duration, cur int, hkr Handshaker, codecFactory CodecFactory) *dl {
	return &dl{
		Dialer: net.Dialer{
			Timeout:   timeout,
			KeepAlive: 5 * time.Second,
		},
		handshaker:   hkr,
		cur:          cur,
		tkt:          ticket.New(cur),
		codecFactory: codecFactory,
	}
}

func (d *dl) dialNode(n *vnode.Node) (p PeerMux, err error) {
	conn, err := d.Dial("tcp", n.Address())
	if err != nil {
		return
	}

	c := d.codecFactory.CreateCodec(conn)
	p, err = d.handshaker.Handshake(c, Inbound)

	return
}
