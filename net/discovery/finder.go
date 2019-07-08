package discovery

import "github.com/vitelabs/go-vite/net/vnode"

type Observer interface {
	Sub(sub Subscriber)
	UnSub(sub Subscriber)
}

type Finder interface {
	Observer
	SetResolver(discv interface {
		GetNodes(count int) (nodes []*vnode.Node)
	})
	FindNeighbors(fromId, target vnode.NodeID, count int) []*vnode.EndPoint
}
