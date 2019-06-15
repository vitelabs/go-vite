package nodemanager

import (
	"github.com/vitelabs/go-vite/node"
	"gopkg.in/urfave/cli.v1"
)

type DefaultNodeManager struct {
	ctx  *cli.Context
	node *node.Node
}

func NewDefaultNodeManager(ctx *cli.Context, maker NodeMaker) (*DefaultNodeManager, error) {
	node, err := maker.MakeNode(ctx)
	if err != nil {
		return nil, err
	}
	return &DefaultNodeManager{
		ctx:  ctx,
		node: node,
	}, nil
}

func (nodeManager *DefaultNodeManager) Start() error {
	// 1: Start up the node
	err := StartNode(nodeManager.node)
	if err != nil {
		return err
	}

	// 2: Waiting for node to close
	WaitNode(nodeManager.node)

	return nil
}

func (nodeManager *DefaultNodeManager) Stop() error {

	StopNode(nodeManager.node)

	return nil
}

func (nodeManager *DefaultNodeManager) Node() *node.Node {

	return nodeManager.node
}
