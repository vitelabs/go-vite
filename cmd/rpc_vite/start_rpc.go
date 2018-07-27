package rpc_vite

import (
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpc/apis"
	"github.com/vitelabs/go-vite/vite"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
)

func StartIpcRpc(vite *vite.Vite, dataDir string) {
	ipcapiURL := filepath.Join(dataDir, rpc.DefaultIpcFile())
	if runtime.GOOS == "windows" {
		ipcapiURL = rpc.DefaultIpcFile()
	}
	println("ipcapiURL: ", ipcapiURL)
	lis, _ := rpc.IpcListen(ipcapiURL)

	exitSig := make(chan os.Signal, 1)
	signal.Notify(exitSig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-exitSig
		println("receiver term sig")
		if lis != nil {
			lis.Close()
		}
	}()

	rpc.StartIPCEndpoint(lis, apis.GetAll(vite))
}
