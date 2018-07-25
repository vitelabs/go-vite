package main

import (
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/p2p"
	"flag"
)

var (
	nameFlag = flag.String("name", "", "boot name")
	sigFlag = flag.String("sig", "", "boot sig")
)


func main ()  {
	flag.Parse()
	p2pConfig := &p2p.Config{
		Name: *nameFlag,
	}

	v, err := vite.New(&vite.Config{
		DataDir:   common.DefaultDataDir(),
		P2pConfig: p2pConfig,
	})


}
