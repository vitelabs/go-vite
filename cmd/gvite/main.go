package main

import (
	"flag"
	"github.com/vitelabs/go-vite"
	"github.com/vitelabs/go-vite/cmd/rpc_vite"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"net/http"
	_ "net/http/pprof"
)

func parseConfig() *config.Config {
	var globalConfig = config.GlobalConfig

	flag.StringVar(&globalConfig.Name, "name", globalConfig.Name, "boot name")
	flag.UintVar(&globalConfig.MaxPeers, "peers", globalConfig.MaxPeers, "max number of connections will be connected")
	flag.StringVar(&globalConfig.Addr, "addr", globalConfig.Addr, "will be listen by vite")
	flag.StringVar(&globalConfig.PrivateKey, "priv", globalConfig.PrivateKey, "hex encode of ed25519 privateKey, use for sign message")
	flag.StringVar(&globalConfig.DataDir, "dir", globalConfig.DataDir, "use for store all files")
	flag.UintVar(&globalConfig.NetID, "netid", globalConfig.NetID, "the network vite will connect")

	flag.Parse()

	globalConfig.P2P.Datadir = globalConfig.DataDir

	return globalConfig
}

func main() {
	govite.PrintBuildVersion()

	mainLog := log15.New("module", "gvite/main")

	go func() {
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			mainLog.Error(err.Error())
		}
	}()

	parsedConfig := parseConfig()

	if s, e := parsedConfig.RunLogDirFile(); e == nil {
		log15.Root().SetHandler(
			log15.LvlFilterHandler(log15.LvlInfo, log15.Must.FileHandler(s, log15.TerminalFormat())),
		)
	}

	vnode, err := vite.New(parsedConfig)

	if err != nil {
		mainLog.Crit("Start vite failed.", "err", err)
	}

	rpc_vite.StartIpcRpcEndpoint(vnode, parsedConfig.DataDir)
}
