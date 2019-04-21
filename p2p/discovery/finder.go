package discovery

import "github.com/vitelabs/go-vite/p2p/vnode"

type Observer interface {
	Sub(sub Subscriber)
	UnSub(sub Subscriber)
}

type Finder interface {
	Observer
	GetNodes(count int) []vnode.Node
}

type closetFinder struct {
	table nodeTable
	subId int
}

func (f *closetFinder) Sub(sub Subscriber) {
	f.subId = sub.Sub(f.receive)
	return
}

func (f *closetFinder) UnSub(sub Subscriber) {
	sub.UnSub(f.subId)
	return
}

func (f *closetFinder) receive(n *vnode.Node) {
	return
}

func (f *closetFinder) GetNodes(count int) []vnode.Node {
	nodes := f.table.nodes(count)
	vnodes := make([]vnode.Node, len(nodes))
	for i, n := range nodes {
		vnodes[i] = n.Node
	}

	return vnodes
}
