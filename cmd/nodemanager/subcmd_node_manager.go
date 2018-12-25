package nodemanager

import (
	"github.com/vitelabs/go-vite/node"
	"gopkg.in/urfave/cli.v1"
)

type SubCmdNodeManager struct {
	ctx  *cli.Context
	node *node.Node
}

func NewSubCmdNodeManager(ctx *cli.Context, maker NodeMaker) (*SubCmdNodeManager, error) {

	node, err := maker.MakeNode(ctx)
	if err != nil {
		return nil, err
	}
	return &SubCmdNodeManager{
		ctx:  ctx,
		node: node,
	}, nil
}

func (nodeManager *SubCmdNodeManager) Start() error {

	// Start up the node
	err := StartNode(nodeManager.node)
	if err != nil {
		return err
	}

	return nil
}

func (nodeManager *SubCmdNodeManager) Stop() error {

	StopNode(nodeManager.node)

	return nil
}

func (nodeManager *SubCmdNodeManager) Node() *node.Node {

	return nodeManager.node
}
