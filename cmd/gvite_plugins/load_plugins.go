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

	"github.com/vitelabs/go-vite/cmd/console"
	"github.com/vitelabs/go-vite/cmd/nodemanager"
	"github.com/vitelabs/go-vite/cmd/params"
	"github.com/vitelabs/go-vite/cmd/utils"
	"github.com/vitelabs/go-vite/log15"
	"gopkg.in/urfave/cli.v1"
)

// gvite is the official command-line client for Vite

var (
	log = log15.New("module", "gvite/main")

	app = cli.NewApp()

	//config
	configFlags = []cli.Flag{
		utils.ConfigFileFlag,
	}
	//general
	generalFlags = []cli.Flag{
		utils.DataDirFlag,
		utils.KeyStoreDirFlag,
	}

	//p2p
	p2pFlags = []cli.Flag{
		utils.DevNetFlag,
		utils.TestNetFlag,
		utils.MainNetFlag,
		utils.IdentityFlag,
		utils.NetworkIdFlag,
		utils.MaxPeersFlag,
		utils.MaxPendingPeersFlag,
		utils.ListenPortFlag,
		utils.NodeKeyHexFlag,
		utils.DiscoveryFlag,
	}

	//IPC
	ipcFlags = []cli.Flag{
		utils.IPCEnabledFlag,
		utils.IPCPathFlag,
	}

	//HTTP RPC
	httpFlags = []cli.Flag{
		utils.RPCEnabledFlag,
		utils.RPCListenAddrFlag,
		utils.RPCPortFlag,
	}

	//WS
	wsFlags = []cli.Flag{
		utils.WSEnabledFlag,
		utils.WSListenAddrFlag,
		utils.WSPortFlag,
	}

	//Console
	consoleFlags = []cli.Flag{
		utils.JSPathFlag,
		utils.ExecFlag,
		utils.PreloadJSFlag,
	}

	//Producer
	producerFlags = []cli.Flag{
		utils.MinerFlag,
		utils.CoinBaseFlag,
		utils.MinerIntervalFlag,
	}

	//Log
	logFlags = []cli.Flag{
		utils.LogLvlFlag,
	}

	//VM
	vmFlags = []cli.Flag{
		utils.VMTestFlag,
		utils.VMTestParamFlag,
	}

	//Net
	netFlags = []cli.Flag{
		utils.SingleFlag,
		utils.FilePortFlag,
	}

	//Stat
	statFlags = []cli.Flag{
		utils.PProfEnabledFlag,
		utils.PProfPortFlag,
	}

	// Metrics
	metricsFlags = []cli.Flag{
		utils.MetricsEnabledFlag,
		utils.InfluxDBEnableFlag,
		utils.InfluxDBEndpointFlag,
		utils.InfluxDBDatabaseFlag,
		utils.InfluxDBUsernameFlag,
		utils.InfluxDBPasswordFlag,
		utils.InfluxDBHostTagFlag,
	}

	// Ledger
	ledgerFlags = []cli.Flag{
		utils.LedgerDeleteToHeight,
		utils.RecoverTrieFlag,
	}

	// Export
	exportFlags = []cli.Flag{
		utils.ExportSbHeightFlags,
	}
)

func init() {

	//TODO: Whether the command name is fixed ？
	app.Name = filepath.Base(os.Args[0])
	app.HideVersion = false
	app.Version = params.Version
	app.Compiled = time.Now()
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "viteLabs",
			Email: "XXX@vite.org",
		},
	}
	app.Copyright = "Copyright 2018-2024 The go-vite Authors"
	app.Usage = "the go-vite cli application"

	//Import: Please add the New command here
	app.Commands = []cli.Command{
		versionCommand,
		licenseCommand,
		consoleCommand,
		attachCommand,
		ledgerRecoverCommand,
		exportCommand,
		pluginDataCommand,
	}
	sort.Sort(cli.CommandsByName(app.Commands))

	//Import: Please add the New Flags here
	app.Flags = utils.MergeFlags(configFlags, generalFlags, p2pFlags,
		ipcFlags, httpFlags, wsFlags, consoleFlags, producerFlags, logFlags,
		vmFlags, netFlags, statFlags, metricsFlags, ledgerFlags, exportFlags)

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
	runtime.GOMAXPROCS(runtime.NumCPU() + 1)

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

	// Resets terminal mode.
	console.Stdin.Close()

	return nil
}
