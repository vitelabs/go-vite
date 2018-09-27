package main

import (
	//_ "net/http/pprof"
	"fmt"
	"github.com/vitelabs/go-vite/cmd/nodemanager"
	"github.com/vitelabs/go-vite/cmd/utils"
	"gopkg.in/urfave/cli.v1"
	"os"
	"sort"
)

// gvite is the official command-line client for Vite

var (
	app = utils.NewApp()

	nodeFlags = []cli.Flag{
		utils.IdentityFlag,
		utils.DataDirFlag,
		utils.NetworkIdFlag,
		utils.ListenPortFlag,
		utils.NodeKeyHexFlag,
		utils.MaxPeersFlag,
		utils.MaxPendingPeersFlag,
		utils.ConfigFileFlag,
	}
)

func init() {
	//Initialize the CLI app and start Gvite
	app.Action = gvite
	//app.HideVersion = true
	app.Copyright = "Copyright 2018-2024 The go-vite Authors"
	app.Commands = []cli.Command{
		initCommand,
		heightCommand,
		versionCommand,
		accountCommand,
		consoleCommand,
		attachCommand,
	}

	sort.Sort(cli.CommandsByName(app.Commands))

	app.Flags = append(app.Flags, nodeFlags...)

	//TODO missing app.Before
	//TODO missing app.After
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func gvite(ctx *cli.Context) error {

	//TODO invalid is why
	fmt.Println(os.Args)

	if args := ctx.Args(); len(args) > 0 {
		return fmt.Errorf("invalid command: %q", args[0])
	}

	nodeManager := nodemanager.NewNodeManager(ctx)

	return nodeManager.Start()
}
