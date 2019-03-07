package discovery

import (
	"github.com/vitelabs/go-vite/p2p2/vnode"
)

type Table interface {
	Add()
	Delete(id vnode.NodeID)
	Active(id vnode.NodeID)
	Closest(id vnode.NodeID) []vnode.NodeID
	Resolve(id vnode.NodeID) (vnode.Node, error)
	Refresh()
	Store()
}

type Sender interface {
	ping()
}

type Finder interface {
}

type Discover interface {
	Start() error
	Stop() error
}

type discovery struct {
	mode   vnode.NodeMode
	finder Finder
}

func New(cfg Config) Discover {

	return nil
}
