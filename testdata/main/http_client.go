package main

import (
	"github.com/vitelabs/go-vite/testdata/main/rpcutils"
	"github.com/vitelabs/go-vite/rpc"
)

func main() {
	client, err := rpc.DialHTTP("http://192.168.31.235:48132")
	if err != nil {
		panic(err)
	}

	rpcutils.Help()
	rpcutils.Cmd(client)
}
