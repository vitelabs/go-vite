package discovery

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/p2p/vnode"
)

func mockAgent(port int, handler func(pkt *packet)) *agent {
	pub, prv, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}

	var id vnode.NodeID
	copy(id[:], pub)
	s := newAgent(prv, &vnode.Node{
		ID: id,
		EndPoint: vnode.EndPoint{
			Host: []byte{127, 0, 0, 1},
			Port: port,
			Typ:  vnode.HostIPv4,
		},
		Net: 2,
		Ext: []byte("hello"),
	}, handler)

	return s
}

func TestSocket_sendNeighbors(t *testing.T) {
	s1 := mockAgent(8483, nil)
	err := s1.start()
	if err != nil {
		panic(err)
	}

	received := make(chan []*vnode.EndPoint)
	s2 := mockAgent(8484, func(pkt *packet) {
		if pkt.c == codeNeighbors {
			ns := pkt.body.(*neighbors)
			fmt.Println("received", len(ns.endpoints), ns.last)
		}
	})
	err = s2.start()
	if err != nil {
		panic(err)
	}

	const total = 1000
	var sent = make([]*vnode.EndPoint, total)
	for i := 0; i < total; i++ {
		sent[i] = &vnode.EndPoint{
			Host: []byte{0, 0, 0, 0},
			Port: i,
			Typ:  vnode.HostIPv4,
		}
	}

	go func() {
		udp, er := net.ResolveUDPAddr("udp", s2.self.Address())
		if er != nil {
			panic(er)
		}

		s2.pool.add(&request{
			expectFrom: s1.self.Address(),
			expectID:   s1.self.ID,
			expectCode: codeNeighbors,
			handler: &findNodeRequest{
				count: total,
				rec:   nil,
				ch:    received,
			},
			expiration: time.Now().Add(expiration * 2),
		})

		er = s1.sendNodes(sent, udp)
		if er != nil {
			panic(er)
		}
	}()

	eps2 := <-received
	if len(eps2) != len(sent) {
		t.Errorf("should not received %d endpoints", len(eps2))
	}

	fmt.Println("maxpayload", maxPayloadLength)
}

func TestSplitEndPoints(t *testing.T) {
	const total = 1000
	var sent = make([]*vnode.EndPoint, total)
	for i := 0; i < total; i++ {
		sent[i] = &vnode.EndPoint{
			Host: []byte{0, 0, 0, 0},
			Port: i,
			Typ:  vnode.HostIPv4,
		}
	}

	ept := splitEndPoints(sent)

	n := &neighbors{
		endpoints: nil,
		last:      false,
		time:      time.Now(),
	}

	pub, prv, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	var id vnode.NodeID
	copy(id[:], pub)
	msg := message{
		c:    codeNeighbors,
		id:   id,
		body: n,
	}

	var i, count int
	for _, eps := range ept {
		count += len(eps)
		for _, ep := range eps {
			if ep != sent[i] {
				t.Fail()
			}
			i++
		}

		n.endpoints = eps
		data, _, e := msg.pack(prv)
		if e != nil {
			panic(e)
		}
		if len(data) > maxPacketLength {
			t.Errorf("too large: %d", len(data))
		} else {
			fmt.Printf("%d bytes udp packet\n", len(data))
		}
	}

	if count != len(sent) {
		t.Errorf("wrong length: %d", count)
	}
}
