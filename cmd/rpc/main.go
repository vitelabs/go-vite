package main

import (
	"github.com/vitelabs/go-vite"
	"github.com/vitelabs/go-vite/cmd/rpc_vite"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	govite.PrintBuildVersion()

	mainLog := log15.New("module", "gvite/main")
	go func() {
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			mainLog.Error(err.Error())
		}
	}()

	if s, e := config.GlobalConfig.RunLogDirFile(); e == nil {
		log15.Root().SetHandler(
			log15.LvlFilterHandler(log15.LvlInfo, log15.Must.FileHandler(s, log15.TerminalFormat())),
		)
	}

	vnode, err := vite.New(config.GlobalConfig)
	if err != nil {
		mainLog.Crit("Start vite failed.", "err", err)
	}

	rpc_vite.StartIpcRpc(vnode, config.GlobalConfig.DataDir)

}
