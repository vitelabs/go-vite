package main

import (
	"context"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/testdata/main/rpcutils"
)

func main() {
	client, err := rpc.DialWebsocket(context.Background(), "ws://localhost:31420", "")
	if err != nil {
		panic(err)
	}

	rpcutils.Help()
	rpcutils.Cmd(client)
}
