package main

import (
	"fmt"
	"github.com/vitelabs/go-vite/cmd/nodemanager"
	"github.com/vitelabs/go-vite/cmd/utils"
	"gopkg.in/urfave/cli.v1"
)

var (
	demoFlags = utils.MergeFlags(configFlags, generalFlags, p2pFlags, ipcFlags, httpFlags, wsFlags, consoleFlags, producerFlags, logFlags, vmFlags, netFlags, statFlags)

	//demo,please add this `demoCommand` to main.go
	/**
	app.Commands = []cli.Command{
		versionCommand,
		licenseCommand,
		consoleCommand,
		attachCommand,
		demoCommand,
	}
	*/
	demoCommand = cli.Command{
		Action:      utils.MigrateFlags(demoAction),
		Name:        "demo",
		Usage:       "demo",
		Flags:       demoFlags,
		Category:    "DEMO COMMANDS",
		Description: `demo`,
	}
)

// localConsole starts a new gvite node, attaching a JavaScript console to it at the same time.
func demoAction(ctx *cli.Context) error {

	// Create and start the node based on the CLI flags
	nodeManager := nodemanager.NewSubCmdNodeManager(ctx, nodemanager.FullNodeMaker{})
	nodeManager.Start()
	defer nodeManager.Stop()

	//Tips: add your code here
	fmt.Println("demo print")
	return nil
}
