package nodemanager

import "github.com/vitelabs/go-vite/node"

type NodeManager interface {
	Start() error

	Stop() error

	Node() *node.Node
}
