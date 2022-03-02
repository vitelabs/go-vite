package subcmd_check_chain

import (
	"fmt"
	"os"

	"gopkg.in/urfave/cli.v1"

	"github.com/vitelabs/go-vite/v2/cmd/nodemanager"
	"github.com/vitelabs/go-vite/v2/cmd/utils"
	"github.com/vitelabs/go-vite/v2/log15"
)

var (
	checkChainCommand = cli.Command{
		Action:   utils.MigrateFlags(checkChainAction),
		Name:     "checkChain",
		Usage:    "checkChain",
		Category: "CHECK CHAIN COMMANDS",
		Flags:    utils.ConfigFlags,
		Description: `
check chain
`,
	}
	log = log15.New("module", "gvite/export")
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
