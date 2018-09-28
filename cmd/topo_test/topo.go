package main

import (
	"encoding/hex"
	"flag"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/topo"
	"log"
	_ "net/http/pprof"
)

func parseConfig() *config.Config {
	var globalConfig = config.GlobalConfig

	flag.StringVar(&globalConfig.Name, "name", globalConfig.Name, "boot name")
	flag.UintVar(&globalConfig.MaxPeers, "peers", globalConfig.MaxPeers, "max number of connections will be connected")
	flag.UintVar(&globalConfig.Port, "port", globalConfig.Port, "will be listen by vite")
	flag.StringVar(&globalConfig.PrivateKey, "priv", globalConfig.PrivateKey, "hex encode of ed25519 privateKey, use for sign message")
	flag.StringVar(&globalConfig.DataDir, "dir", globalConfig.DataDir, "use for store all files")
	flag.UintVar(&globalConfig.NetID, "netid", globalConfig.NetID, "the network vite will connect")

	flag.Parse()

	globalConfig.P2P.Datadir = globalConfig.DataDir

	return globalConfig
}

func main() {
	parsedConfig := parseConfig()

	p2pCfg := parsedConfig.P2P

	cfg := p2p.Config{
		Name:            p2pCfg.Name,
		NetID:           p2p.NetworkID(p2pCfg.NetID),
		MaxPeers:        p2pCfg.MaxPeers,
		MaxPendingPeers: uint(p2pCfg.MaxPendingPeers),
		MaxInboundRatio: p2pCfg.MaxPassivePeersRatio,
		Port:            uint(p2pCfg.Port),
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
						_, err := rw.ReadMsg()
						if err != nil {
							return err
						}
					}
					return nil
				},
			},
		},
		BootNodes: p2pCfg.BootNodes,
	}

	if p2pCfg.PrivateKey != "" {
		priv, err := hex.DecodeString(p2pCfg.PrivateKey)
		if err == nil {
			cfg.PrivateKey = ed25519.PrivateKey(priv)
		}
	}

	svr, err := p2p.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	th, err := topo.New(parsedConfig.Topo)
	if err != nil {
		log.Fatal(err)
	}
	th.Start(svr)
	defer th.Stop()

	svr.Protocols = append(svr.Protocols, th.Protocol())

	err = svr.Start()
	if err != nil {
		log.Fatal(err)
	}
	defer svr.Stop()
	//err = http.ListenAndServe("localhost:6060", nil)
	//if err != nil {
	//	log.Println("pprof server error: ", err)
	//}
	pending := make(chan struct{})
	<-pending
}
