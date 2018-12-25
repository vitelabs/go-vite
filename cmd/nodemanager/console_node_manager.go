package nodemanager

import (
	"github.com/vitelabs/go-vite/node"
	"gopkg.in/urfave/cli.v1"
)

type ConsoleNodeManager struct {
	ctx  *cli.Context
	node *node.Node
}

func NewConsoleNodeManager(ctx *cli.Context, maker NodeMaker) (*ConsoleNodeManager, error) {
	node, err := maker.MakeNode(ctx)
	if err != nil {
		return nil, err
	}
	return &ConsoleNodeManager{
		ctx:  ctx,
		node: node,
	}, nil
}

func (nodeManager *ConsoleNodeManager) Start() error {

	// Start up the node
	err := StartNode(nodeManager.node)
	if err != nil {
		return err
	}

	return nil
}

func (nodeManager *ConsoleNodeManager) Stop() error {

	StopNode(nodeManager.node)

	return nil
}

func (nodeManager *ConsoleNodeManager) Node() *node.Node {

	return nodeManager.node
}
