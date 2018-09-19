package main

import (
	"github.com/vitelabs/go-vite/cmd/utils"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/node"
	"gopkg.in/urfave/cli.v1"
)

func makeConfigNode(ctx *cli.Context) (*node.Node, *config.Config) {

	//Load defaults
	//inside create default config and try load config from json file
	cfg := config.GlobalConfig

	//Config log to file
	if fileName, e := cfg.RunLogDirFile(); e == nil {
		log15.Root().SetHandler(
			log15.LvlFilterHandler(log15.LvlInfo, log15.Must.FileHandler(fileName, log15.TerminalFormat())),
		)
	}

	// Apply flags
	utils.SetNodeConfig(ctx, cfg)
	node, err := node.New(cfg)

	if err != nil {
		log15.Error("Failed to create the node: %v", err)
	}

	//TODO future will Apply other flags

	return node, cfg
}

func makeFullNode(ctx *cli.Context) *node.Node {
	node, _ := makeConfigNode(ctx)
	return node
}
