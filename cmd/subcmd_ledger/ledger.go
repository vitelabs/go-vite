package subcmd_ledger

import (
	"gopkg.in/urfave/cli.v1"

	"github.com/vitelabs/go-vite"
	"github.com/vitelabs/go-vite/cmd/nodemanager"
	"github.com/vitelabs/go-vite/cmd/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
)

var (
	QueryLedgerCommand = cli.Command{
		Name:        "ledger",
		Usage:       "ledger accounts",
		Category:    "LOCAL COMMANDS",
		Description: `Load ledger.`,
		Subcommands: []cli.Command{
			{
				Name:  "dumpbalances",
				Usage: "dump all accounts balance",
				Flags: append(utils.ConfigFlags, []cli.Flag{
					cli.StringFlag{
						Name:  "tokenId",
						Usage: "tokenId",
					},
					cli.Uint64Flag{
						Name:  "snapshotHeight",
						Usage: "snapshot height",
					}}...),
				Action: utils.MigrateFlags(dumpAllBalanceAction),
			},
		},
	}
	log = log15.New("module", "ledger")
)

func localVite(ctx *cli.Context) (*vite.Vite, error) {
	node, err := nodemanager.LocalNodeMaker{}.MakeNode(ctx)
	if err != nil {
		return nil, err
	}

	if err := node.Prepare(); err != nil {
		return nil, err
	}
	node.Vite().Chain()
	return node.Vite(), nil
}

func dumpAllBalanceAction(ctx *cli.Context) error {
	vite, err := localVite(ctx)
	if err != nil {
		return err
	}
	tokenId, err := types.HexToTokenTypeId(ctx.String("tokenId"))
	if err != nil {
		return err
	}
	snapshotHeight := ctx.Uint64("snapshotHeight")
	return dumpBalance(vite.Chain(), tokenId, snapshotHeight)
}
