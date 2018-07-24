package main

import (
	"fmt"
	"github.com/vitelabs/go-vite/common"
	vrpc "github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/wallet"
	"path/filepath"
	"strings"
	"github.com/vitelabs/go-vite/vite"
)

func main() {
	vite.New(vite.NewP2pConfig())
	m := wallet.NewManager(filepath.Join(common.TestDataDir(), "wallet"))
	rpcAPI := []vrpc.API{
		{
			Namespace: "wallet",
			Public:    true,
			Service:   m,
			Version:   "1.0"},
	}

	handler, err := vrpc.StartInProc(rpcAPI)

	client := vrpc.DialInProc(handler)

	var s string
	err = client.Call("wallet.ListAddress", nil, &s)
	if err != nil {
		print(err.Error())
	}
	fmt.Printf("list(%v):\n", len(strings.Split(s, "\n")))
	println(s)
}
