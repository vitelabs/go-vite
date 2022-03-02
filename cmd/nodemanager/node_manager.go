package nodemanager

import "github.com/vitelabs/go-vite/v2/node"

type NodeManager interface {
	Start() error

	Stop() error

	Node() *node.Node
}
