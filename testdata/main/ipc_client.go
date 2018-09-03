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
	//rpcutils.Unlock(client, "vite_269ecd4bef9cef499e991eb9667ec4a33cfdfed832c8123ada", "123456", "0")

	rpcutils.Cmd(client)
}
