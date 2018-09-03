package rpc_vite

import (
	"fmt"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi"
	"github.com/vitelabs/go-vite/vite"
	"path/filepath"
)

func StartRpc(vite *vite.Vite, dataDir string) {
	ipcapiURL := filepath.Join(dataDir, common.DefaultIpcFile())
	fmt.Println(ipcapiURL)
	go rpc.StartIPCEndpoint(ipcapiURL, rpcapi.GetAllApis(vite))
	go rpc.StartWSEndpoint(common.DefaultWSEndpoint(), rpcapi.GetAllApis(vite), nil, []string{"*"}, true)
	c := make(chan int)
	<-c
}
