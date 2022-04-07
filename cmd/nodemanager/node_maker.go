package nodemanager

import (
	"gopkg.in/urfave/cli.v1"

	"github.com/vitelabs/go-vite/v2/node"
	nodeconfig "github.com/vitelabs/go-vite/v2/node/config"
)

type NodeMaker interface {

	//create Node
	MakeNode(ctx *cli.Context) (*node.Node, error)

	//create NodeConfig
	MakeNodeConfig(ctx *cli.Context) (*nodeconfig.Config, error)
}
