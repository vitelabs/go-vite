package main

import (
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpc/apis"
	"github.com/vitelabs/go-vite/vite"
	"log"
	"path/filepath"
	"runtime"
)

func main() {
	cfg := vite.DefaultConfig
	vnode, err := vite.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	ipcapiURL := filepath.Join(cfg.DataDir, rpc.DefaultIpcFile())
	if runtime.GOOS == "windows" {
		ipcapiURL = rpc.DefaultIpcFile()
	}
	lis, err := rpc.IpcListen(ipcapiURL)
	defer func() {
		if lis != nil {
			lis.Close()
		}
	}()

	rpc.StartIPCEndpoint(lis, apis.GetAll(vnode))

}
