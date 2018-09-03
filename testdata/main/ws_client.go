package main

import (
	"context"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/testdata/main/rpcutils"
)

func main() {
	client, err := rpc.DialWebsocket(context.Background(), common.DefaultWSEndpoint(), "")
	if err != nil {
		panic(err)
	}
	rpcutils.Help()
	//rpcutils.Unlock(client, "vite_269ecd4bef9cef499e991eb9667ec4a33cfdfed832c8123ada", "123456", "0")

	rpcutils.Cmd(client)
}
