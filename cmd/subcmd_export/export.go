package subcmd_export

import (
	"fmt"
	"os"

	"gopkg.in/urfave/cli.v1"

	"github.com/vitelabs/go-vite/v2/cmd/nodemanager"
	"github.com/vitelabs/go-vite/v2/cmd/utils"
	"github.com/vitelabs/go-vite/v2/log15"
)

var (
	ExportCommand = cli.Command{
		Action:   utils.MigrateFlags(exportLedgerAction),
		Name:     "export",
		Usage:    "export --sbHeight=5000000",
		Flags:    append(utils.ExportFlags, utils.ConfigFlags...),
		Category: "EXPORT COMMANDS",
		Description: `
Export ledger.
`,
	}
	log = log15.New("module", "gvite/export")
)

func exportLedgerAction(ctx *cli.Context) error {
	// Create and start the node based on the CLI flags
	nodeManager, err := nodemanager.NewExportNodeManager(ctx, nodemanager.FullNodeMaker{})
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
