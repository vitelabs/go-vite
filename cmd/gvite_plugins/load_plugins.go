package gvite_plugins

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"gopkg.in/urfave/cli.v1"

	"github.com/vitelabs/go-vite/v2/cmd/nodemanager"
	"github.com/vitelabs/go-vite/v2/cmd/subcmd_export"
	"github.com/vitelabs/go-vite/v2/cmd/subcmd_ledger"
	"github.com/vitelabs/go-vite/v2/cmd/subcmd_loadledger"
	"github.com/vitelabs/go-vite/v2/cmd/subcmd_plugin_data"
	"github.com/vitelabs/go-vite/v2/cmd/subcmd_recover"
	"github.com/vitelabs/go-vite/v2/cmd/subcmd_rpc"
	"github.com/vitelabs/go-vite/v2/cmd/subcmd_virtualnode"
	"github.com/vitelabs/go-vite/v2/cmd/utils"
	"github.com/vitelabs/go-vite/v2/log15"
	"github.com/vitelabs/go-vite/v2/version"
)

// gvite is the official command-line client for Vite

var (
	log = log15.New("module", "gvite/main")

	app = cli.NewApp()
)

func init() {

	//TODO: Whether the command name is fixed ？
	app.Name = filepath.Base(os.Args[0])
	app.HideVersion = false
	app.Version = version.VITE_BUILD_VERSION
	app.Compiled = time.Now()
	app.Authors = []cli.Author{
		{
			Name:  "Vite Labs",
			Email: "info@vite.org",
		},
	}
	app.Copyright = "Copyright 2018-2024 The go-vite Authors"
	app.Usage = "the go-vite cli application"

	//Import: Please add the New command here
	app.Commands = []cli.Command{
		versionCommand,
		licenseCommand,
		subcmd_recover.LedgerRecoverCommand,
		subcmd_export.ExportCommand,
		subcmd_plugin_data.PluginDataCommand,
		subcmd_rpc.RpcCommand,
		subcmd_loadledger.LoadLedgerCommand,
		subcmd_ledger.QueryLedgerCommand,
		subcmd_virtualnode.VirtualNodeCommand,
	}
	sort.Sort(cli.CommandsByName(app.Commands))

	//Import: Please add the New Flags here
	for _, element := range app.Commands {
		app.Flags = utils.MergeFlags(app.Flags, element.Flags)
	}
	app.Flags = utils.MergeFlags(app.Flags, utils.StatFlags)

	app.Before = beforeAction
	app.Action = action
	app.After = afterAction
}

func Loading() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func beforeAction(ctx *cli.Context) error {
	max := runtime.NumCPU() + 1
	log.Info("runtime num", "max", max)
	runtime.GOMAXPROCS(max)

	//TODO: we can add dashboard here
	if ctx.GlobalIsSet(utils.PProfEnabledFlag.Name) {
		pprofPort := ctx.GlobalUint(utils.PProfPortFlag.Name)
		var listenAddress string
		if pprofPort == 0 {
			pprofPort = 8080
		}
		listenAddress = fmt.Sprintf("%s:%d", "0.0.0.0", pprofPort)
		var visitAddress = fmt.Sprintf("http://localhost:%d/debug/pprof", pprofPort)

		go func() {
			log.Info("Enable chain performance analysis tool, you can visit the address of `" + visitAddress + "`")
			http.ListenAndServe(listenAddress, nil)
		}()
	}

	return nil
}

func action(ctx *cli.Context) error {

	//Make sure No subCommands were entered,Only the flags
	if args := ctx.Args(); len(args) > 0 {
		return fmt.Errorf("invalid command: %q", args[0])
	}

	nodeManager, err := nodemanager.NewDefaultNodeManager(ctx, nodemanager.FullNodeMaker{})
	if err != nil {
		return fmt.Errorf("new node error, %+v", err)
	}

	return nodeManager.Start()
}

func afterAction(ctx *cli.Context) error {
	return nil
}
