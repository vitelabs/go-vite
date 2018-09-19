package main

import (
	"encoding/hex"
	"flag"
	"github.com/vitelabs/go-vite"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"net"
	"net/http"
	_ "net/http/pprof"
	"time"
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

	p2pCfg := parsedConfig.P2P

	addr, _ := net.ResolveTCPAddr("tcp", p2pCfg.Addr)

	cfg := p2p.Config{
		Name:            p2pCfg.Name,
		NetID:           p2p.NetworkID(p2pCfg.NetID),
		MaxPeers:        p2pCfg.MaxPeers,
		MaxPendingPeers: uint(p2pCfg.MaxPendingPeers),
		MaxInboundRatio: p2pCfg.MaxPassivePeersRatio,
		Port:            uint(addr.Port),
		Database:        p2pCfg.Datadir,
		PrivateKey:      nil,
		Protocols: []*p2p.Protocol{
			{
				Name: "p2p-test",
				// use for message command set, should be unique
				ID: 1,
				// read and write Msg with rw
				Handle: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
					for {
						time.Sleep(time.Hour)
					}
					return nil
				},
			},
		},
		BootNodes: []string{
			"vnode://33e43481729850fc66cef7f42abebd8cb2f1c74f0b09a5bf03da34780a0a5606@150.109.120.109:8483",
			"vnode://7194af5b7032cb470c41b313e2675e2c3ba3377e66617247012b8d638552fb17@150.109.17.20:8483",
			"vnode://087c45631c3ec9a5dbd1189084ee40c8c4c0f36731ef2c2cb7987da421d08ba9@162.62.21.17:8483",
			"vnode://7c6a2b920764b6dddbca05bb6efa1c9bcd90d894f6e9b107f137fc496c802346@170.106.1.142:8483",
			"vnode://2840979ae06833634764c19e72e6edbf39595ff268f558afb16af99895aba3d8@49.51.168.181:8483",
		},
	}

	if p2pCfg.PrivateKey != "" {
		priv, err := hex.DecodeString(p2pCfg.PrivateKey)
		if err == nil {
			cfg.PrivateKey = ed25519.PrivateKey(priv)
		}
	}

	svr := p2p.New(cfg)

	pending := make(chan struct{})
	svr.Start()

	<-pending
}
