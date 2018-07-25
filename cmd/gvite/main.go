package main

import (
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/p2p"
	"flag"
	"log"
)

var (
	nameFlag = flag.String("name", "", "boot name")
	sigFlag = flag.String("sig", "", "boot sig")
)


func main ()  {
	flag.Parse()
	p2pConfig := &p2p.Config{
		CmdConfig: p2p.CmdConfig{
			Name: *nameFlag,
			Sig: *sigFlag,
		},
	}

	v, err := vite.New(&vite.Config{
		DataDir:   common.DefaultDataDir(),
		P2pConfig: p2pConfig,
	})

	if err != nil {
		log.Fatal(err)
	}
}
