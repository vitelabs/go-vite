package rpc_vite

import (
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vrpc"
	"github.com/vitelabs/go-vite/vite"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
)

func StartIpcRpc(vite *vite.Vite, dataDir string) {
	ipcapiURL := filepath.Join(dataDir, vrpc.DefaultIpcFile())
	if runtime.GOOS == "windows" {
		ipcapiURL = vrpc.DefaultIpcFile()
	}
	log15.Root().Info("StartIpcRpc ", "ipcapiURL: ", ipcapiURL)
	lis, _ := vrpc.IpcListen(ipcapiURL)

	exitSig := make(chan os.Signal, 1)
	signal.Notify(exitSig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-exitSig
		log15.Root().Info("receiver term sig")
		if lis != nil {
			lis.Close()
		}
	}()

	vrpc.StartIPCEndpoint(lis, vrpc.GetAllApis(vite))
}
