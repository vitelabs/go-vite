package p2p

import (
	"net"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/tools/ticket"

	"github.com/vitelabs/go-vite/p2p2/vnode"
)

type dialer interface {
	dialNode(node vnode.Node) (Peer, error)
	dialAddress(address string) (p Peer, err error)
}

const maxDialCount = 3
const retryDialDuration = 30 * time.Second

type dialTask struct {
	node  vnode.Node
	times int
	err   error
	next  *dialTask
}

type dialTasks struct {
	rw sync.RWMutex
	m  map[string]struct{}
	l  *dialTask
	t  *dialTask
}

func (ts *dialTasks) has(address string) bool {
	ts.rw.RLock()
	defer ts.rw.RUnlock()

	_, ok := ts.m[address]
	return ok
}

func (ts *dialTasks) append() {

}

type dl struct {
	d     net.Dialer
	hk    handshaker
	tasks map[string]struct{} // use address as key
	cur   int
	tkt   ticket.Ticket
}

func newDialer(timeout time.Duration, cur int, hk handshaker) dialer {
	return &dl{
		d: net.Dialer{
			Timeout: timeout,
		},
		hk:    hk,
		tasks: make(map[string]struct{}),
		cur:   cur,
		tkt:   ticket.New(cur),
	}
}

func (d *dl) dialNode(node vnode.Node) (p Peer, err error) {
	conn, err := d.d.Dial("tcp", node.Host())
	if err != nil {
		return
	}

	//p, err = d.hk.handshake(conn, true)

	return
}

func (d *dl) dialAddress(address string) (p Peer, err error) {

}
