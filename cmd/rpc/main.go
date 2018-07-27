package main

import (
	"github.com/vitelabs/go-vite/cmd/rpc_vite"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/vite"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	config.RecoverConfig()
	vnode, err := vite.New(config.GlobalConfig)
	if err != nil {
		log.Fatal(err)
	}

	rpc_vite.StartIpcRpc(vnode, config.GlobalConfig.DataDir)

}
