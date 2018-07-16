package main

import (
	"bufio"
	"fmt"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/wallet"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	fmt.Println("Enter d for Default or any others for Test ")
	inputReader := bufio.NewReader(os.Stdin)
	input, err := inputReader.ReadString('\n')
	dir := common.TestDataDir()
	if strings.HasPrefix(input, "d") {
		dir = common.DefaultDataDir()
	}

	ipcapiURL := filepath.Join(dir, rpc.DefaultIpcFile())
	if runtime.GOOS == "windows" {
		ipcapiURL = rpc.DefaultIpcFile()
	}

	var wapi wallet.JsonApi
	m := wallet.NewManager(filepath.Join(dir, "wallet"))
	wapi = m.JsonApi

	rpcAPI := []rpc.API{
		{
			Namespace: "wallet",
			Public:    true,
			Service:   wapi,
			Version:   "1.0"},
	}

	lis, err := rpc.IpcListen(ipcapiURL)
	defer func() {
		if lis != nil {
			lis.Close()
		}
	}()
	go rpc.StartIPCEndpoint(lis, rpcAPI)

	inputReader = bufio.NewReader(os.Stdin)
	fmt.Println("Enter any key to stop ")
	input, err = inputReader.ReadString('\n')
	if err == nil {
		fmt.Printf("The input was: %s\n", input)
	}

}
