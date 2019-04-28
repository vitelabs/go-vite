package net

import (
	"sync"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"github.com/vitelabs/go-vite/p2p/vnode"
)

const extensionLen = 96

type sbpCollector struct {
	producer Producer
	rw       sync.RWMutex
	sbps     map[types.Address]*vnode.Node
	subId    int
}

func (s *sbpCollector) GetNodes(count int) []vnode.Node {
	s.rw.RLock()
	defer s.rw.RUnlock()

	nodes := make([]vnode.Node, len(s.sbps))
	var i int
	for _, n := range s.sbps {
		nodes[i] = *n
		i++
	}

	return nodes
}

func (s *sbpCollector) receive(n *vnode.Node) {
	if len(n.Ext) < extensionLen {
		return
	}

	pub := n.Ext[:32]
	sign := n.Ext[32:extensionLen]
	if ed25519.Verify(pub, n.ID.Bytes(), sign) {
		address := types.PubkeyToAddress(pub)
		if s.producer.IsProducer(address) {
			s.rw.Lock()
			s.sbps[address] = n
			s.rw.Unlock()
		}
	}
}

func (s *sbpCollector) Sub(sub discovery.Subscriber) {
	s.subId = sub.Sub(s.receive)
}

func (s *sbpCollector) UnSub(sub discovery.Subscriber) {
	sub.UnSub(s.subId)
}
