package subcmd_recover

import (
	"fmt"
	"os"

	"gopkg.in/urfave/cli.v1"

	"github.com/vitelabs/go-vite/v2/cmd/nodemanager"
	"github.com/vitelabs/go-vite/v2/cmd/utils"
	"github.com/vitelabs/go-vite/v2/log15"
)

var (
	LedgerRecoverCommand = cli.Command{
		Action:    utils.MigrateFlags(recoverLedgerAction),
		Name:      "recover",
		Usage:     "recover --del=500000",
		ArgsUsage: "--del=500000",
		Flags:     append(utils.LedgerFlags, utils.ConfigFlags...),
		Category:  "RECOVER COMMANDS",
		Description: `
Recover ledger.
`,
	}
	log = log15.New("module", "gvite/recover")
)

func recoverLedgerAction(ctx *cli.Context) error {
	// Create and start the node based on the CLI flags
	nodeManager, err := nodemanager.NewRecoverNodeManager(ctx, nodemanager.FullNodeMaker{})
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
