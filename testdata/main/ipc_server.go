package main

import (
	"bufio"
	"fmt"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/wallet"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	fmt.Println("Enter D for Default or any others for Test ")
	inputReader := bufio.NewReader(os.Stdin)
	input, err := inputReader.ReadString('\n')
	dir := common.TestDataDir()
	if strings.HasPrefix(input, "D") {
		dir = common.DefaultDataDir()
	}

	ipcapiURL := filepath.Join(dir, "unixrpc.ipc")
	m := wallet.NewManager(filepath.Join(dir, "wallet"))
	m.Init()

	rpcAPI := []rpc.API{
		{
			Namespace: "wallet",
			Public:    true,
			Service:   m,
			Version:   "1.0"},
	}
	listener, _, err := rpc.StartIPCEndpoint(ipcapiURL, rpcAPI)
	defer func() {
		if listener != nil {
			listener.Close()
		}
	}()
	if err != nil {
		panic(err.Error())
	}

	inputReader = bufio.NewReader(os.Stdin)
	fmt.Println("Enter any key to stop ")
	input, err = inputReader.ReadString('\n')
	if err == nil {
		fmt.Printf("The input was: %s\n", input)
	}

}
