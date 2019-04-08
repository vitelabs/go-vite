package discovery

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

func TestPool_add(t *testing.T) {
	var p = newRequestPool()

	p.start()

	p.add(&request{
		expectFrom: "127.0.0.1:8483",
		expectID:   vnode.NodeID{},
		expectCode: 0,
		handler:    nil,
		expiration: time.Now().Add(time.Hour),
	})

	p.add(&request{
		expectFrom: "127.0.0.1:8483",
		expectID:   vnode.NodeID{},
		expectCode: 0,
		handler:    nil,
		expiration: time.Now().Add(time.Hour),
	})

	p.add(&request{
		expectFrom: "127.0.0.1:8484",
		expectID:   vnode.NodeID{},
		expectCode: 0,
		handler:    nil,
		expiration: time.Now().Add(time.Hour),
	})

	if p.size() != 3 {
		t.Errorf("should be %d request", 3)
	}
}

func TestPool_rec(t *testing.T) {
	var p = newRequestPool()

	p.start()

	ch := make(chan *Node)
	p.add(&request{
		expectFrom: "127.0.0.1:8483",
		expectID:   vnode.ZERO,
		expectCode: codePong,
		handler: &pingRequest{
			hash: []byte("hello"),
			done: ch,
		},
		expiration: time.Now().Add(time.Second),
	})

	go p.rec(&packet{
		message: message{
			c:  codePong,
			id: vnode.NodeID{1},
			body: &pong{
				from: &vnode.EndPoint{
					Host: []byte{127, 0, 0, 1},
					Port: 8888,
					Typ:  vnode.HostIPv4,
				},
				to:   nil,
				net:  0,
				ext:  []byte("world"),
				echo: []byte("hello"),
				time: time.Now(),
			},
		},
		from: &net.UDPAddr{
			IP:   []byte{127, 0, 0, 1},
			Port: 8483,
			Zone: "",
		},
	})

	node := <-ch
	if node == nil {
		t.Error("should not be nil")
		return
	}
	if node.ID[0] != 1 {
		t.Error("wrong id")
	}
	if !bytes.Equal(node.Ext, []byte("world")) {
		t.Error("wrong ext")
	}
	if node.Address() != "127.0.0.1:8888" {
		t.Errorf("wrong address: %s", node.Address())
	}
}

func TestPool_rec2(t *testing.T) {
	var p = newRequestPool()

	p.start()

	const total = 1000
	received := make(chan []*vnode.EndPoint)
	p.add(&request{
		expectFrom: "127.0.0.1:8483",
		expectID:   vnode.ZERO,
		expectCode: codeNeighbors,
		handler: &findNodeRequest{
			count: total,
			rec:   nil,
			ch:    received,
		},
		expiration: time.Now().Add(time.Second),
	})

	var eps = make([]*vnode.EndPoint, total)
	for i := 0; i < total; i++ {
		eps[i] = &vnode.EndPoint{
			Host: []byte{127, 0, 0, 1},
			Port: i,
			Typ:  vnode.HostIPv4,
		}
	}

	go func() {
		ept := splitEndPoints(eps)

		for i, epl := range ept {
			p.rec(&packet{
				message: message{
					c:  codeNeighbors,
					id: vnode.ZERO,
					body: &neighbors{
						endpoints: epl,
						last:      i == len(ept)-1,
						time:      time.Now(),
					},
				},
				from: &net.UDPAddr{
					IP:   []byte{127, 0, 0, 1},
					Port: 8483,
					Zone: "",
				},
			})
		}
	}()

	eps2 := <-received
	if len(eps2) != len(eps) {
		t.Errorf("should not be %d endpoints", len(eps2))
		return
	}
}
