package main

import (
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpc/apis"
	"github.com/vitelabs/go-vite/vite"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
)

func main() {
	config.RecoverConfig()
	vnode, err := vite.New(config.GlobalConfig)
	if err != nil {
		log.Fatal(err)
	}

	ipcapiURL := filepath.Join(config.GlobalConfig.DataDir, rpc.DefaultIpcFile())
	if runtime.GOOS == "windows" {
		ipcapiURL = rpc.DefaultIpcFile()
	}
	println("ipcapiURL: ", ipcapiURL)
	lis, err := rpc.IpcListen(ipcapiURL)

	exitSig := make(chan os.Signal, 1)
	signal.Notify(exitSig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-exitSig
		println("receiver term sig")
		if lis != nil {
			lis.Close()
		}
	}()

	rpc.StartIPCEndpoint(lis, apis.GetAll(vnode))

}
