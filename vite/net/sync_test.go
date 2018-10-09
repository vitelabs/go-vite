package net

import (
	"flag"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	config2 "github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/p2p"
	"strings"
	"testing"
)

var config = p2p.Config{
	Name:            "",
	NetID:           10,
	MaxPeers:        100,
	MaxPendingPeers: 20,
	MaxInboundRatio: 3,
	Port:            8483,
	PrivateKey:      nil,
	BootNodes:       nil,
}

var privateKey string

func init() {
	var bootNodes string

	flag.StringVar(&config.Name, "name", "net_test", "server name")
	flag.StringVar(&privateKey, "private key", "", "server private key")
	flag.StringVar(&bootNodes, "boot nodes", "", "boot nodes")

	config.BootNodes = strings.Split(bootNodes, ",")

	flag.Parse()
}

func TestNet(t *testing.T) {
	chainInstance := chain.NewChain(&config2.Config{
		DataDir: common.DefaultDataDir(),
	})

	chainInstance.Init()
	chainInstance.Start()

	net, err := New(&Config{
		Port:     8484,
		Chain:    chainInstance,
		Verifier: nil,
	})

	if err != nil {
		t.Error(err)
	}

	svr, err := p2p.New(config)
	if err != nil {
		t.Error(err)
	}

	svr.Protocols = append(svr.Protocols, net.Protocols...)

	err = svr.Start()
	if err != nil {
		t.Error(err)
	}
}
