package gvite_plugins

import (
	"fmt"
	"github.com/vitelabs/go-vite/cmd/nodemanager"
	"github.com/vitelabs/go-vite/cmd/utils"
	"gopkg.in/urfave/cli.v1"
	"os"
)

var (
	checkChainCommand = cli.Command{
		Action:   utils.MigrateFlags(checkChainAction),
		Name:     "checkChain",
		Usage:    "checkChain",
		Category: "CHECK CHAIN COMMANDS",
		Flags:    configFlags,
		Description: `
check chain
`,
	}
)

func checkChainAction(ctx *cli.Context) error {
	// Create and start the node based on the CLI flags
	nodeManager, err := nodemanager.NewCheckChainNodeManager(ctx, nodemanager.FullNodeMaker{})
	if err != nil {
		log.Error(fmt.Sprintf("new Node error, %+v", err))
		return err
	}
	if err := nodeManager.Start(); err != nil {
		log.Error(err.Error())
		fmt.Println(err.Error())
		return err
	}

	os.Exit(0)
	return nil
}
