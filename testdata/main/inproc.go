package main

import (
	"fmt"
	"github.com/vitelabs/go-vite/common"
	vrpc "github.com/vitelabs/go-vite/vrpc"
	"github.com/vitelabs/go-vite/wallet"
	"path/filepath"
	"strings"
)

func main() {
	m := wallet.NewManagerAndInit(filepath.Join(common.GoViteTestDataDir(), "wallet"))
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
