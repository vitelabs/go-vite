package discovery

import "github.com/vitelabs/go-vite/p2p/vnode"

type Finder interface {
	GetNodes(count int) []vnode.Node
	Observer
}

type closetFinder struct {
	table nodeTable
}

func (f *closetFinder) Sub(Subscriber) {
	return
}

func (f *closetFinder) UnSub(Subscriber) {
	return
}

func (f *closetFinder) Receive(n *vnode.Node) {
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
