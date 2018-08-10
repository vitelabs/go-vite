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

var (
	nameFlag       = flag.String("name", "", "boot name")
	maxPeersFlag   = flag.Uint("maxpeers", 0, "max number of connections will be connected")
	addrFlag       = flag.String("addr", "0.0.0.0:8483", "will be listen by vite")
	privateKeyFlag = flag.String("priv", "", "use for sign message")
	dataDirFlag    = flag.String("dir", "", "use for store all files")
	netIdFlag      = flag.Uint("netid", 0, "the network vite will connect")
	//minerFlag     = flag.Bool("miner", false, "boot miner")
	//minerInterval = flag.Int("minerInterval", 6, "miner interval(unit sec).")
	//coinbaseFlag  = flag.String("coinbaseAddress", "", "boot coinbaseAddress")
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

	flag.Parse()

	globalConfig := config.GlobalConfig

	if *dataDirFlag != "" {
		globalConfig.DataDir = *dataDirFlag
	}

	globalConfig.P2P = config.MergeP2PConfig(&config.P2P{
		Name:       *nameFlag,
		MaxPeers:   uint32(*maxPeersFlag),
		Addr:       *addrFlag,
		PrivateKey: *privateKeyFlag,
		NetID:      *netIdFlag,
	})
	globalConfig.P2P.Datadir = globalConfig.DataDir

	//globalConfig.Miner = config.MergeMinerConfig(&config.Miner{
	//	Miner:         *minerFlag,
	//	Coinbase:      *coinbaseFlag,
	//	MinerInterval: *minerInterval,
	//})

	if s, e := config.GlobalConfig.RunLogDirFile(); e == nil {
		log15.Root().SetHandler(
			log15.LvlFilterHandler(log15.LvlInfo, log15.Must.FileHandler(s, log15.TerminalFormat())),
		)
	}

	vnode, err := vite.New(globalConfig)

	if err != nil {
		mainLog.Crit("Start vite failed.", "err", err)
	}

	rpc_vite.StartIpcRpc(vnode, globalConfig.DataDir)
}
