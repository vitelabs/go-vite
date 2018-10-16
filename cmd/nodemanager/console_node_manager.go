package nodemanager

import (
	"github.com/vitelabs/go-vite/node"
	"gopkg.in/urfave/cli.v1"
)

type ConsoleNodeManager struct {
	ctx  *cli.Context
	node *node.Node
}

func NewConsoleNodeManager(ctx *cli.Context, maker NodeMaker) ConsoleNodeManager {
	return ConsoleNodeManager{
		ctx:  ctx,
		node: maker.MakeNode(ctx),
	}
}

func (nodeManager *ConsoleNodeManager) Start() error {

	// Start up the node
	StartNode(nodeManager.node)

	return nil
}

func (nodeManager *ConsoleNodeManager) Stop() error {

	StopNode(nodeManager.node)

	return nil
}

func (nodeManager *ConsoleNodeManager) Node() *node.Node {

	return nodeManager.node
}
