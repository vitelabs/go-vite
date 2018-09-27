package main

import (
	"encoding/hex"
	"flag"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"log"
	"net"
	_ "net/http/pprof"
	"strconv"
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

	addr, _ := net.ResolveUDPAddr("udp", "0.0.0.0:" + strconv.FormatUint(uint64(p2pCfg.Port), 10))

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
	}

	var nodes []*discovery.Node
	for _, str := range p2pCfg.BootNodes {
		n, err := discovery.ParseNode(str)
		if err == nil {
			nodes = append(nodes, n)
		}
	}


	var priv ed25519.PrivateKey

	if p2pCfg.PrivateKey != "" {
		priv, err = hex.DecodeString(p2pCfg.PrivateKey)
		if err == nil {
			priv = ed25519.PrivateKey(priv)
		}
	} else {
		log.Fatal("privateKey is nil")
	}

	ID, _ := discovery.Priv2NodeID(priv)

	discvCfg := &discovery.Config{
		Priv:      priv,
		DBPath:    "",
		BootNodes: nodes,
		Conn:      conn,
		Self:      &discovery.Node{
			ID:  ID,
			IP:  addr.IP,
			UDP: uint16(addr.Port),
			TCP: uint16(addr.Port),
		},
	}

	svr := discovery.New(discvCfg)

	svr.Start()

	//err = http.ListenAndServe("localhost:6060", nil)
	//if err != nil {
	//	log.Println("pprof server error: ", err)
	//}
	pending := make(chan struct{})
	<-pending
}
