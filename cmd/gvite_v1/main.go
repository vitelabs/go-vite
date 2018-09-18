package main

import (
	//_ "net/http/pprof"
	"fmt"
	"github.com/vitelabs/go-vite/cmd/rpc_vite"
	"github.com/vitelabs/go-vite/cmd/utils"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"gopkg.in/urfave/cli.v1"
	"net/http"
	"os"
	"sort"
)

// gvite is the official command-line client for Vite
const (
	ClientIdentifier = "gvite"
)

var (
	app = utils.NewApp()

	nodeFlags = []cli.Flag{
		utils.DataDirFlag,
		utils.KeystoreDirFlag,
	}
)

func init() {
	//Initialize the CLI app and start Gvite
	app.Action = gvite
	//app.HideVersion = true
	app.Copyright = "Copyright 2018-2024 The go-vite Authors"
	app.Commands = []cli.Command{
		// ledgercmd.go
		//initCommand,
		heightCommand,

		// aidcmd.go
		versionCommand,

		// accountcmd.go
		accountCommand,

		// consolecmd.go
		consoleCommand,
		attachCommand,
	}

	sort.Sort(cli.CommandsByName(app.Commands))

	app.Flags = append(app.Flags, nodeFlags...)
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func gvite(ctx *cli.Context) error {

	fmt.Println("Hello Vite!")
	mainLog := log15.New("module", "gvite/main")

	go func() {
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			mainLog.Error(err.Error())
		}
	}()

	//localConfig := makeConfigNode()
	localConfig := config.GlobalConfig
	vnode, err := vite.New(localConfig)

	if err != nil {
		mainLog.Crit("Start vite failed.", "err", err)
	}

	rpc_vite.StartIpcRpcEndpoint(vnode, localConfig.DataDir)

	return nil
}
