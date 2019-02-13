package gvite_plugins

import (
	"fmt"
	"github.com/vitelabs/go-vite/cmd/nodemanager"
	"github.com/vitelabs/go-vite/cmd/utils"
	"gopkg.in/urfave/cli.v1"
	"os"
)

var (
	ledgerRecoverCommand = cli.Command{
		Action:    utils.MigrateFlags(recoverLedgerAction),
		Name:      "recover",
		Usage:     "Recover ledger",
		ArgsUsage: "--del=500000",
		Flags:     ledgerFlags,
		Category:  "RECOVER COMMANDS",
		Description: `
Recover ledger.
`,
	}
)

func recoverLedgerAction(ctx *cli.Context) error {
	// Create and start the node based on the CLI flags
	nodeManager, err := nodemanager.NewRecoverNodeManager(ctx, nodemanager.FullNodeMaker{})
	if err != nil {
		log.Error(fmt.Sprintf("new Node error, %+v", err))
		return err
	}
	nodeManager.Start()
	os.Exit(0)
	return nil
}
