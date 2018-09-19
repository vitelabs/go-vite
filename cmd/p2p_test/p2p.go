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
		BootNodes: nil,
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
