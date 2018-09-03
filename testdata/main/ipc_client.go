package main

// this is a rpc client for go-vite debug .

import (
	"context"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/rpc"
	"path/filepath"
	"github.com/vitelabs/go-vite/testdata/main/rpcutils"
)

func main() {
	ipcapiURL := filepath.Join(common.DefaultDataDir(), rpc.DefaultIpcFile())
	client, err := rpc.DialIPC(context.Background(), ipcapiURL)
	if err != nil {
		panic(err)
	}
	rpcutils.Help()

	rpcutils.Cmd(client)
}
