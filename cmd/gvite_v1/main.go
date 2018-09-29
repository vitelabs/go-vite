package main

import (
	//_ "net/http/pprof"
	"fmt"
	"github.com/vitelabs/go-vite/cmd/console"
	"github.com/vitelabs/go-vite/cmd/nodemanager"
	"github.com/vitelabs/go-vite/cmd/params"
	"github.com/vitelabs/go-vite/cmd/utils"
	"github.com/vitelabs/go-vite/log15"
	"gopkg.in/urfave/cli.v1"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"
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
		//misc
		initCommand,
		versionCommand,
		//console
		//consoleCommand,
		//attachCommand,
	}
	sort.Sort(cli.CommandsByName(app.Commands))

	//Import: Please add the New Flags here
	app.Flags = utils.MergeFlags(configFlags, generalFlags, p2pFlags, ipcFlags, httpFlags, wsFlags, consoleFlags)

	app.Before = beforeAction
	app.Action = action
	app.After = afterAction
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func beforeAction(ctx *cli.Context) error {
	runtime.GOMAXPROCS(runtime.NumCPU())

	//TODO: we can add dashboard here

	return nil
}

func action(ctx *cli.Context) error {

	//Make sure No subCommands were entered,Only the flags
	if args := ctx.Args(); len(args) > 0 {
		return fmt.Errorf("invalid command: %q", args[0])
	}

	nodeManager := nodemanager.New(ctx, nodemanager.FullNodeMaker{})

	return nodeManager.Start()
}

func afterAction(ctx *cli.Context) error {

	// Resets terminal mode.
	console.Stdin.Close()

	return nil
}
