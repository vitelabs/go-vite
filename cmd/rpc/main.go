package main

import (
	"github.com/vitelabs/go-vite/cmd/rpc_vite"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/vite"
	"log"
	"net/http"
	_ "net/http/pprof"
	"github.com/inconshreveable/log15"
	"github.com/vitelabs/go-vite/common/types"
)

func main() {
	addresses, _, _ := types.CreateAddress()

	log1 := log15.New("module", "l1")
	//log1.Info("message",  errors.New("adsa"))
	log1.Info("message", "err", addresses)
	//log2 := log1.New("parter", "l2")
	//log2.Info("log2")
	//log3 := log1.New("l3")
	//log3.Info("llll")

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
