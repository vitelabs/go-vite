package nodemanager

import (
	"github.com/vitelabs/go-vite/node"
	"gopkg.in/urfave/cli.v1"
)

type NodeMaker interface {

	//create Node
	MakeNode(ctx *cli.Context) *node.Node

	//create NodeConfig
	MakeNodeConfig(ctx *cli.Context) *node.Config
}
