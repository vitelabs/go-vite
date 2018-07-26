package main

import (
	"github.com/vitelabs/go-vite/cmd/rpc_vite"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/vite"
	"log"
)

func main() {
	config.RecoverConfig()
	vnode, err := vite.New(config.GlobalConfig)
	if err != nil {
		log.Fatal(err)
	}

	rpc_vite.StartIpcRpc(vnode, config.GlobalConfig.DataDir)

}
