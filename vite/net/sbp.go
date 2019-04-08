package net

import (
	"github.com/vitelabs/go-vite/p2p/discovery"
	"github.com/vitelabs/go-vite/p2p/vnode"
)

type sbpCollector struct {
	producer Producer
	sbps     map[vnode.NodeID]*vnode.Node
}

func (s *sbpCollector) GetNodes(count int) []vnode.Node {
	panic("implement me")
}

func (s *sbpCollector) Receive(n *vnode.Node) {

}

func (s *sbpCollector) Sub(discovery.Subscriber) {
	panic("implement me")
}

func (s *sbpCollector) UnSub(discovery.Subscriber) {
	panic("implement me")
}
