package main

import (
	"fmt"
	"github.com/vitelabs/go-vite/cmd/nodemanager"
	"github.com/vitelabs/go-vite/cmd/utils"
	"gopkg.in/urfave/cli.v1"
)

var (
	demoFlags = utils.MergeFlags(configFlags, generalFlags, p2pFlags, ipcFlags, httpFlags, wsFlags, consoleFlags, producerFlags, logFlags, vmFlags, netFlags, statFlags)

	//demo
	demoCommand = cli.Command{
		Action:   utils.MigrateFlags(demoAction),
		Name:     "demo",
		Usage:    "demo",
		Flags:    jsFlags,
		Category: "DEMO COMMANDS",
		Description: `
The GVite console is an interactive shell for the JavaScript runtime environment
which exposes a node admin interface as well as the Ðapp JavaScript API.
See https://github.com/vitelabs/go-vite/wiki/JavaScript-Console.`,
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
