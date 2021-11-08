package subcmd_ledger

import (
	"gopkg.in/urfave/cli.v1"

	"github.com/vitelabs/go-vite"
	"github.com/vitelabs/go-vite/cmd/nodemanager"
	"github.com/vitelabs/go-vite/cmd/utils"
)

var (
	QueryLedgerCommand = cli.Command{
		Name:        "ledger",
		Usage:       "ledger accounts",
		Flags:       utils.ConfigFlags,
		Category:    "LOCAL COMMANDS",
		Description: `Load ledger.`,
		Subcommands: []cli.Command{
			{
				Name:   "dumpbalances",
				Usage:  "dump all accounts balance",
				Action: utils.MigrateFlags(dumpAllBalanceAction),
			},
		},
	}
)

func localVite(ctx *cli.Context) (*vite.Vite, error) {
	node, err := nodemanager.LocalNodeMaker{}.MakeNode(ctx)
	if err != nil {
		return nil, err
	}

	if err := node.Prepare(); err != nil {
		return nil, err
	}
	return node.Vite(), nil
}

func dumpAllBalanceAction(ctx *cli.Context) error {
	_, err := localVite(ctx)
	if err != nil {
		return err
	}

	return nil
}
