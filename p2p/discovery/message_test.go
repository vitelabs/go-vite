package discovery

import (
	"bytes"
	"crypto/rand"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/p2p/vnode"
)

func equalPing(p, p2 *ping) bool {
	if p.from != nil {
		if !p.from.Equal(p2.from) {
			return false
		}
	} else if p2.from != nil {
		return false
	}

	if p.to != nil {
		if !p.to.Equal(p2.to) {
			return false
		}
	} else if p2.to != nil {
		return false
	}

	if p.net != p2.net {
		return false
	}
	if !bytes.Equal(p.ext, p2.ext) {
		return false
	}
	if p.time.Unix() != p2.time.Unix() {
		return false
	}
	return true
}

func TestPing_Serialize(t *testing.T) {
	var p = &ping{
		from: nil,
		to:   nil,
		net:  0,
		ext:  []byte("hello"),
		time: time.Time{}.Add(time.Hour),
	}

	data, err := p.serialize()
	if err != nil {
		t.Errorf("failed to serialize ping: %v", err)
	}

	var p2 = new(ping)
	err = p2.deserialize(data)
	if err != nil {
		t.Errorf("failed to deserialize ping: %v", err)
	}

	if !equalPing(p, p2) {
		t.Error("not equal")
	}

	// init endpoint
	p.from = &vnode.EndPoint{}
	p.to = &vnode.EndPoint{}
	data, err = p.serialize()
	if err != nil {
		t.Errorf("failed to serialize ping: %v", err)
	}

	err = p2.deserialize(data)
	if err != nil {
		t.Errorf("failed to deserialize ping: %v", err)
	}
	if p2.from != nil || p2.to != nil {
		t.Error("endpoint should be nil")
	}
}

func equalPong(p, p2 *pong) bool {
	if p.from != nil {
		if !p.from.Equal(p2.from) {
			return false
		}
	} else if p2.from != nil {
		return false
	}

	if p.to != nil {
		if !p.to.Equal(p2.to) {
			return false
		}
	} else if p2.to != nil {
		return false
	}

	if p.net != p2.net {
		return false
	}
	if !bytes.Equal(p.ext, p2.ext) {
		return false
	}
	if !bytes.Equal(p.echo, p2.echo) {
		return false
	}
	if p.time.Unix() != p2.time.Unix() {
		return false
	}
	return true
}

func TestPong_Serialize(t *testing.T) {
	var p = &pong{
		echo: []byte("hello"),
		from: nil,
		to:   nil,
		net:  0,
		ext:  []byte("hello"),
		time: time.Time{}.Add(time.Hour),
	}

	data, err := p.serialize()
	if err != nil {
		t.Errorf("failed to serialize ping: %v", err)
	}

	var p2 = new(pong)
	err = p2.deserialize(data)
	if err != nil {
		t.Errorf("failed to deserialize ping: %v", err)
	}

	if !equalPong(p, p2) {
		t.Error("not equal")
	}

	// init endpoint
	p.from = &vnode.EndPoint{}
	p.to = &vnode.EndPoint{}
	data, err = p.serialize()
	if err != nil {
		t.Errorf("failed to serialize ping: %v", err)
	}

	err = p2.deserialize(data)
	if err != nil {
		t.Errorf("failed to deserialize ping: %v", err)
	}
	if p2.from != nil || p2.to != nil {
		t.Error("endpoint should be nil")
	}
}

func equalFindNode(p, p2 *findnode) bool {
	if p.count != p2.count {
		return false
	}
	if p.target != p2.target {
		return false
	}
	if p.time.Unix() != p2.time.Unix() {
		return false
	}
	return true
}

func TestFindNode_Serialize(t *testing.T) {
	var p = &findnode{
		count:  10,
		target: vnode.ZERO,
		time:   time.Time{}.Add(time.Hour),
	}
	_, _ = rand.Read(p.target.Bytes())

	data, err := p.serialize()
	if err != nil {
		t.Errorf("failed to serialize ping: %v", err)
	}

	var p2 = new(findnode)
	err = p2.deserialize(data)
	if err != nil {
		t.Errorf("failed to deserialize ping: %v", err)
	}

	if !equalFindNode(p, p2) {
		t.Error("not equal")
	}
}

func equalNeighbors(p, p2 *neighbors) bool {
	if p.last != p2.last {
		return false
	}
	if len(p.endpoints) != len(p2.endpoints) {
		return false
	}
	for i, ep := range p.endpoints {
		ep2 := p2.endpoints[i]
		if !ep.Equal(ep2) {
			return false
		}
	}
	if p.time.Unix() != p2.time.Unix() {
		return false
	}
	return true
}

func TestNeighbors_Serialize(t *testing.T) {
	var p = &neighbors{
		endpoints: nil,
		last:      true,
		time:      time.Time{}.Add(time.Hour),
	}
	p.endpoints = append(p.endpoints, &vnode.EndPoint{
		Host: []byte{0, 0, 0, 0},
		Port: 8888,
		Typ:  vnode.HostIPv4,
	}, &vnode.EndPoint{
		Host: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		Port: 8889,
		Typ:  vnode.HostIPv6,
	}, &vnode.EndPoint{
		Host: []byte("vite.org"),
		Port: 9000,
		Typ:  vnode.HostDomain,
	})

	data, err := p.serialize()
	if err != nil {
		t.Errorf("failed to serialize ping: %v", err)
	}

	var p2 = new(neighbors)
	err = p2.deserialize(data)
	if err != nil {
		t.Errorf("failed to deserialize ping: %v", err)
	}

	if !equalNeighbors(p, p2) {
		t.Error("not equal")
	}
}

func equalMessage(m1, m2 message) bool {
	if m1.c != m2.c {
		return false
	}
	if m1.id != m2.id {
		return false
	}
	if m1.c == codePing {
		p1 := m1.body.(*ping)
		p2 := m2.body.(*ping)
		return equalPing(p1, p2)
	}
	if m1.c == codePong {
		p1 := m1.body.(*pong)
		p2 := m2.body.(*pong)
		return equalPong(p1, p2)
	}
	if m1.c == codeFindnode {
		p1 := m1.body.(*findnode)
		p2 := m2.body.(*findnode)
		return equalFindNode(p1, p2)
	}
	if m1.c == codeNeighbors {
		p1 := m1.body.(*neighbors)
		p2 := m2.body.(*neighbors)
		return equalNeighbors(p1, p2)
	}

	return false
}

func TestMessage_Pack(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Error(err)
	}

	id, _ := vnode.Bytes2NodeID(publicKey)

	messages := []message{
		{
			c:  codePing,
			id: id,
			body: &ping{
				from: &vnode.EndPoint{
					Host: []byte{127, 0, 0, 1},
					Port: 8483,
					Typ:  vnode.HostIPv4,
				},
				to: &vnode.EndPoint{
					Host: []byte{127, 0, 0, 1},
					Port: 8484,
					Typ:  vnode.HostIPv4,
				},
				net:  1,
				ext:  []byte("hello world"),
				time: time.Time{}.Add(time.Hour),
			},
		},
		{
			c:  codePong,
			id: id,
			body: &pong{
				from: &vnode.EndPoint{
					Host: []byte{127, 0, 0, 1},
					Port: 8483,
					Typ:  vnode.HostIPv4,
				},
				to: &vnode.EndPoint{
					Host: []byte{127, 0, 0, 1},
					Port: 8484,
					Typ:  vnode.HostIPv4,
				},
				net:  1,
				ext:  []byte("hello world"),
				time: time.Time{}.Add(2 * time.Hour),
				echo: []byte("world"),
			},
		},
		{
			c:  codeFindnode,
			id: id,
			body: &findnode{
				target: vnode.ZERO,
				count:  100,
				time:   time.Time{}.Add(3 * time.Hour),
			},
		},
		{
			c:  codeNeighbors,
			id: id,
			body: &neighbors{
				endpoints: []*vnode.EndPoint{
					{
						Host: []byte("vite.org"),
						Port: 8888,
						Typ:  vnode.HostDomain,
					},
				},
				last: true,
				time: time.Time{}.Add(time.Hour),
			},
		},
	}

	var data, hash []byte
	var p *packet
	for _, msg := range messages {
		data, hash, err = msg.pack(privateKey)
		if err != nil {
			t.Error(err)
		}

		p, err = unPacket(data)
		if err != nil {
			t.Error(err)
		}

		if !equalMessage(p.message, msg) {
			t.Error("not equal message")
		}

		if !bytes.Equal(p.hash, hash) {
			t.Error("not equal hash")
		}
	}
}

func TestUnpack(t *testing.T) {
	var buf []byte
	for i := 0; i < minPacketLength; i++ {
		_, _ = unPacket(buf)
		buf = append(buf, 1)
	}
	_, _ = unPacket(buf)
}
