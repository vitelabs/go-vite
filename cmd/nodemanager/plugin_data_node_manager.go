package nodemanager

import (
	"github.com/vitelabs/go-vite/node"
	"gopkg.in/urfave/cli.v1"
)

type PluginDataNodeManager struct {
	ctx  *cli.Context
	node *node.Node
}

func NewPluginDataNodeManager(ctx *cli.Context, maker NodeMaker) (*PluginDataNodeManager, error) {
	node, err := maker.MakeNode(ctx)
	if err != nil {
		return nil, err
	}

	// single mode
	node.Config().Single = true
	node.ViteConfig().Net.Single = true

	// no miner
	node.Config().MinerEnabled = false
	node.ViteConfig().Producer.Producer = false

	// no ledger gc
	ledgerGc := false
	node.Config().LedgerGc = &ledgerGc
	node.ViteConfig().Chain.LedgerGc = ledgerGc

	// open plugins
	openPlugins := true
	node.Config().OpenPlugins = &openPlugins
	node.ViteConfig().Chain.OpenPlugins = openPlugins

	return &PluginDataNodeManager{
		ctx:  ctx,
		node: node,
	}, nil
}

func (nodeManager *PluginDataNodeManager) Start() error {
	node := nodeManager.node

	err := StartNode(nodeManager.node)
	if err != nil {
		return err
	}

	c := node.Vite().Chain()
	if err := c.Plugins().RebuildData(); err != nil {
		return err
	}
	return nil
}

func (nodeManager *PluginDataNodeManager) Stop() error {

	StopNode(nodeManager.node)

	return nil
}

func (nodeManager *PluginDataNodeManager) Node() *node.Node {
	return nodeManager.node
}
