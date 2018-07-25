package main

import (
	"flag"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite"
	"log"
)

var (
	nameFlag = flag.String("name", "", "boot name")
	sigFlag  = flag.String("sig", "", "boot sig")
)

func main() {
	flag.Parse()
	p2pConfig := &p2p.Config{
		CmdConfig: p2p.CmdConfig{
			Name: *nameFlag,
			Sig:  *sigFlag,
		},
	}

	_, err := vite.New(&vite.Config{
		DataDir:   common.DefaultDataDir(),
		P2pConfig: p2pConfig,
	})

	if err != nil {
		log.Fatalf("Start vue failed. Error is %v\n", err)
	}
}
