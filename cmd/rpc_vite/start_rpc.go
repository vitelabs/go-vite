package rpc_vite

import (
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/rpcapi"
	"github.com/vitelabs/go-vite/vite"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
)

func StartIpcRpc(vite *vite.Vite, dataDir string) {
	ipcapiURL := filepath.Join(dataDir, rpcapi.DefaultIpcFile())
	if runtime.GOOS == "windows" {
		ipcapiURL = rpcapi.DefaultIpcFile()
	}
	log15.Root().Info("StartIpcRpc ", "ipcapiURL: ", ipcapiURL)
	lis, _ := rpcapi.IpcListen(ipcapiURL)

	exitSig := make(chan os.Signal, 1)
	signal.Notify(exitSig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-exitSig
		log15.Root().Info("receiver term sig")
		if lis != nil {
			lis.Close()
		}
	}()

	rpcapi.StartIPCEndpoint(lis, rpcapi.GetAllApis(vite))
}
