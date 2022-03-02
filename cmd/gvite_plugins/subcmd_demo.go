package gvite_plugins

import (
	"fmt"

	"gopkg.in/urfave/cli.v1"

	"github.com/vitelabs/go-vite/v2/cmd/nodemanager"
	"github.com/vitelabs/go-vite/v2/cmd/utils"
)

var (
	demoFlags = utils.MergeFlags(utils.ConfigFlags, utils.GeneralFlags)

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

// localConsole starts chain new gvite node, attaching chain JavaScript console to it at the same time.
func demoAction(ctx *cli.Context) error {

	// Create and start the node based on the CLI flags
	nodeManager, err := nodemanager.NewSubCmdNodeManager(ctx, nodemanager.FullNodeMaker{})
	if err != nil {
		fmt.Println("demo error", err)
		return err
	}
	nodeManager.Start()
	defer nodeManager.Stop()

	//Tips: add your code here
	fmt.Println("demo print")
	return nil
}
