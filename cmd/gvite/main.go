package main

import (
	"flag"
	"github.com/inconshreveable/log15"
	"github.com/vitelabs/go-vite/cmd/rpc_vite"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/vite"
	"net/http"
	_ "net/http/pprof"
)

var (
	nameFlag  = flag.String("name", "", "boot name")
	sigFlag   = flag.String("sig", "", "boot sig")
	maxPeers  = flag.Uint("maxpeers", 0, "max number of connections will be connected")
	passRatio = flag.Uint("passration", 0, "max passive connections will be connected")

	minerFlag     = flag.Bool("miner", false, "boot miner")
	minerInterval = flag.Int("minerInterval", 6, "miner interval(unit sec).")
	coinbaseFlag  = flag.String("coinbaseAddress", "", "boot coinbaseAddress")
)

func main() {
	mainLog := log15.New("module", "gvite/main")

	go func() {
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			mainLog.Info(err.Error())
		}
	}()

	flag.Parse()

	globalConfig := config.GlobalConfig

	globalConfig.P2P = config.MergeP2PConfig(&config.P2P{
		Name:                 *nameFlag,
		Sig:                  *sigFlag,
		MaxPeers:             uint32(*maxPeers),
		MaxPassivePeersRatio: uint32(*passRatio),
	})
	globalConfig.P2P.Datadir = globalConfig.DataDir

	globalConfig.Miner = config.MergeMinerConfig(&config.Miner{
		Miner:         *minerFlag,
		Coinbase:      *coinbaseFlag,
		MinerInterval: *minerInterval,
	})

	globalConfig.ConfigureLog()

	vnode, err := vite.New(globalConfig)

	if err != nil {
		mainLog.Crit("Start vite failed.", "err", err)
	}

	rpc_vite.StartIpcRpc(vnode, globalConfig.DataDir)
}
