package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/rpc"
	rpc2 "net/rpc"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	ipcapiURL := filepath.Join(common.TestDataDir(), "unixrpc.ipc")
	client, err := rpc.DialIPC(context.Background(), ipcapiURL)
	if err != nil {
		panic(err)
	}

	list(client)
	fmt.Println("input ls to show addressed\ninput 'ca' to create a address with password 123456  ")

	for {
		inputReader := bufio.NewReader(os.Stdin)
		input, err := inputReader.ReadString('\n')

		if err != nil {
			return
		}
		if strings.HasPrefix(input, "quit") {
			return
		}
		if strings.HasPrefix(input, "ls") {
			list(client)
		} else if strings.HasPrefix(input, "ca") {
			createAddress(client, "123456")
		} else {
			fmt.Printf("The input was: %s\n", input)
		}
	}

}

func list(client *rpc2.Client) error {
	var s string
	err := client.Call("wallet.ListAddress", nil, &s)
	if err != nil {
		print(err.Error())
	}
	fmt.Printf("list(%v):\n", len(strings.Split(s, "\n")))
	println(s)
	return err
}

func createAddress(client *rpc2.Client, pwd string) {
	var s string
	err := client.Call("wallet.NewAddress", [...]string{pwd}, &s)
	if err != nil {
		println(err.Error())
	}
	println("create success " + s)
}
