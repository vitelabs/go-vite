package main

import (
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/node"
	"github.com/vitelabs/go-vite/vite"
	"gopkg.in/urfave/cli.v1"
)

func makeConfigNode() *config.Config {

	//TODO

	var localconfig = config.GlobalConfig

	if s, e := localconfig.RunLogDirFile(); e == nil {
		log15.Root().SetHandler(
			log15.LvlFilterHandler(log15.LvlInfo, log15.Must.FileHandler(s, log15.TerminalFormat())),
		)
	}

	return localconfig
}

func makeFullNode(ctx *cli.Context) (*node.Node, error) {
	cfg := makeConfigNode()
	vnode, err := vite.New(cfg)
	if err != nil {
		return nil, err
	}
	//return vnode, nil
	return nil, nil
}
