package discovery

import "github.com/vitelabs/go-vite/net/vnode"

type Observer interface {
	Sub(sub Subscriber)
	UnSub(sub Subscriber)
}

type Finder interface {
	Observer
	SetResolver(discv interface {
		ReadNodes(n int) []*vnode.Node
	})
}
