package subcmd_ledger

import (
	"fmt"

	"gopkg.in/urfave/cli.v1"

	"github.com/vitelabs/go-vite/v2"
	"github.com/vitelabs/go-vite/v2/cmd/nodemanager"
	"github.com/vitelabs/go-vite/v2/cmd/utils"
	"github.com/vitelabs/go-vite/v2/common"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/log15"
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
			{
				Name:   "latestBlock",
				Usage:  "print latest snapshot block",
				Flags:  utils.ConfigFlags,
				Action: utils.MigrateFlags(latestSnapshotBlockAction),
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

func latestSnapshotBlockAction(ctx *cli.Context) error {
	vite, err := localVite(ctx)
	if err != nil {
		return err
	}
	block := vite.Chain().GetLatestSnapshotBlock()

	fmt.Println(common.ToJson(block))
	return nil
}
