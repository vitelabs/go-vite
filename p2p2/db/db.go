package db

import (
	"time"

	"github.com/vitelabs/go-vite/p2p2/vnode"
)

type NodeDB interface {
	Close() error
	Store(n *vnode.Node)
	Retrieve(id vnode.NodeID)
	Delete(id vnode.NodeID)
	RetrieveSeed(deadline time.Time) []*vnode.Node
}
