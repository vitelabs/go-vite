package rpc_vite

import (
	"fmt"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi"
	"github.com/vitelabs/go-vite/vite"
	"path/filepath"
)

func StartAllRpcEndpoint(vite *vite.Vite, dataDir string) {
	ipcapiURL := filepath.Join(dataDir, common.DefaultIpcFile())
	fmt.Println(ipcapiURL)
	go rpc.StartIPCEndpoint(ipcapiURL, rpcapi.GetAllApis(vite))
	go rpc.StartWSEndpoint(common.DefaultWSEndpoint(), rpcapi.GetPublicApis(vite), nil, []string{"*"}, true)
	go rpc.StartHTTPEndpoint(common.DefaultHttpEndpoint(), rpcapi.GetPublicApis(vite), nil, []string{"*"}, nil, rpc.DefaultHTTPTimeouts)
	c := make(chan int)
	<-c
}

func StartIpcRpcEndpoint(vite *vite.Vite, dataDir string) {
	ipcapiURL := filepath.Join(dataDir, common.DefaultIpcFile())
	fmt.Println(ipcapiURL)
	go rpc.StartIPCEndpoint(ipcapiURL, rpcapi.GetAllApis(vite))
	c := make(chan int)
	<-c
}
