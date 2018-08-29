package main

import (
	//_ "net/http/pprof"
	"os"
	"fmt"
	"github.com/vitelabs/go-vite/cmd/utils"
	"gopkg.in/urfave/cli.v1"
	"sort"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/log15"
	"net/http"
	"github.com/vitelabs/go-vite/cmd/rpc_vite"
	"github.com/vitelabs/go-vite/config"
)

//func parseConfig() *config.Config {
//	var globalConfig = config.GlobalConfig
//
//	flag.StringVar(&globalConfig.Name, "name", globalConfig.Name, "boot name")
//	flag.UintVar(&globalConfig.MaxPeers, "peers", globalConfig.MaxPeers, "max number of connections will be connected")
//	flag.StringVar(&globalConfig.Addr, "addr", globalConfig.Addr, "will be listen by vite")
//	flag.StringVar(&globalConfig.PrivateKey, "priv", globalConfig.PrivateKey, "hex encode of ed25519 privateKey, use for sign message")
//	flag.StringVar(&globalConfig.DataDir, "dir", globalConfig.DataDir, "use for store all files")
//	flag.UintVar(&globalConfig.NetID, "netid", globalConfig.NetID, "the network vite will connect")
//
//	flag.Parse()
//
//	globalConfig.P2P.Datadir = globalConfig.DataDir
//
//	return globalConfig
//}

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

func init()  {
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

	rpc_vite.StartIpcRpc(vnode, localConfig.DataDir)

	return nil
}
