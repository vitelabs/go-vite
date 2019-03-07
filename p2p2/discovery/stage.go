package discovery

import (
	"sync"

	"github.com/vitelabs/go-vite/p2p2/vnode"
)

type stage struct {
	nodes map[vnode.NodeID][]endPoint
	mu    sync.Mutex
}
