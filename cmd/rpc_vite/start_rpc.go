package rpc_vite

import (
	"github.com/vitelabs/go-vite/rpcapi"
	"github.com/vitelabs/go-vite/vite"
	"path/filepath"
	"github.com/vitelabs/go-vite/rpc"
	"fmt"
)

func StartIpcRpc(vite *vite.Vite, dataDir string) {
	rpc.BlockMode = true
	ipcapiURL := filepath.Join(dataDir, rpc.DefaultIpcFile())
	fmt.Println(ipcapiURL)
	rpc.StartIPCEndpoint(ipcapiURL, rpcapi.GetAllApis(vite))
}
