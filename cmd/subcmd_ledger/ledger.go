package subcmd_ledger

import (
	"fmt"

	"gopkg.in/urfave/cli.v1"

	"github.com/vitelabs/go-vite/cmd/nodemanager"
	"github.com/vitelabs/go-vite/cmd/utils"
	"github.com/vitelabs/go-vite/common/types"
)

var (
	QueryLedgerCommand = cli.Command{
		Action:      utils.MigrateFlags(queryLedgerAction),
		Name:        "ledger",
		Usage:       "ledger accounts",
		Flags:       utils.ConfigFlags,
		Category:    "LOCAL COMMANDS",
		Description: `Load ledger.`,
	}
)

func queryLedgerAction(ctx *cli.Context) error {
	node, err := nodemanager.LocalNodeMaker{}.MakeNode(ctx)
	if err != nil {
		return err
	}

	if err := node.Prepare(); err != nil {
		return err
	}
	ch := node.Vite().Chain()
	ch.IterateAccounts(func(addr types.Address, accountId uint64, err error) bool {
		if err == nil {
			fmt.Println(addr.String())
		}
		return true
	})
	return nil
}
