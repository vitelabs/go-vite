package nodemanager

import (
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/node"
	"gopkg.in/urfave/cli.v1"
)

var (
	log = log15.New("module", "gvite/nodemanager")
)

type NodeManager struct {
	ctx  *cli.Context
	node *node.Node
}

func New(ctx *cli.Context, maker NodeMaker) NodeManager {
	return NodeManager{
		ctx:  ctx,
		node: maker.MakeNode(ctx),
	}
}

func (nodeManager *NodeManager) Start() error {

	// Start up the node
	StartNode(nodeManager.node)
	// Waiting for node to close
	WaitNode(nodeManager.node)
	return nil
}

func (nodeManager *NodeManager) Stop() error {

	StopNode(nodeManager.node)
	return nil
}

func (nodeManager *NodeManager) Node() *node.Node {
	return nodeManager.node
}
