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
					},
					cli.StringFlag{
						Name:  "minAirdrop",
						Usage: "only airdrop above (full decimals)",
						Value: "100000000000000", // 0.0001 for 18 decimals
					},
					cli.BoolFlag{
						Name:  "allowContract",
						Usage: "allow smart contract addresses",
					},
					cli.BoolFlag{
						Name:  "showBuiltInContract",
						Usage: "show built-in smart contracts",
					},
					&cli.StringFlag{
						Name:  "ignoreList",
						Usage: "ignore addresses feed in a file",
						Value: "ignore.txt",
					},
					cli.StringFlag{
						Name:  "airdropSum",
						Usage: "total airdrop amount (full decimals)",
						Value: "0",
					},cli.StringFlag{
						Name:  "quotaSum",
						Usage: "total airdrop for staking quota (full decimals)",
						Value: "0",
					},}...),
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
	minAirdrop := ctx.String("minAirdrop")
	allowContract := ctx.Bool("allowContract")
	showBuiltIn := ctx.Bool("showBuiltInContract")
	ignoreList := ctx.String("ignoreList")
	airdropSum := ctx.String("airdropSum")
	quotaSum := ctx.String("quotaSum")
	return dumpBalance(vite.Chain(), tokenId, snapshotHeight, minAirdrop, allowContract, showBuiltIn, ignoreList, airdropSum, quotaSum)
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
