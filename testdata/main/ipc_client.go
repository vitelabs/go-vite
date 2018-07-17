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
	client, err := rpc.DialIPC(context.Background(), ipcapiURL)
	if err != nil {
		panic(err)
	}

	list(client)
	fmt.Println("input List to show addressed\ninput 'Create' to create a address with password 123456  ")

	for {
		inputReader := bufio.NewReader(os.Stdin)
		input, err := inputReader.ReadString('\n')

		if err != nil {
			return
		}
		if strings.HasPrefix(input, "quit") {
			return
		}
		if strings.HasPrefix(input, "List") {
			list(client)
		} else if strings.HasPrefix(input, "Create") {
			createAddress(client, "123456")
		} else if strings.HasPrefix(input, "Status") {
			status(client)
		} else if strings.HasPrefix(input, "UnLock") {
			param := strings.Split(strings.TrimRight(input, "\n"), " ")[1:]
			Unlock(client, param)
		} else if strings.HasPrefix(input, "SignDataWithPassphrase") {
			println("SignDataWithPassphrase")
			param := strings.Split(strings.TrimRight(input, "\n"), " ")[1:]
			SignDataWithPassphrase(client, param)
		} else if strings.HasPrefix(input, "SignData") {
			param := strings.Split(strings.TrimRight(input, "\n"), " ")[1:]
			SignData(client, param)
		} else if strings.HasPrefix(input, "Lock") {
			param := strings.Split(strings.TrimRight(input, "\n"), " ")[1:]
			Lock(client, param)
		} else if strings.HasPrefix(input, "ImportPriv") {
			param := strings.Split(strings.TrimRight(input, "\n"), " ")[1:]
			ImportPriv(client, param)
		} else if strings.HasPrefix(input, "ExportPriv") {
			param := strings.Split(strings.TrimRight(input, "\n"), " ")[1:]
			ExportPriv(client, param)
		} else {
			fmt.Printf("The input was: %s\n", input)
		}
	}

}

func list(client *rpc2.Client) {
	doRpcCall(client, "wallet.ListAddress", nil)
}

func createAddress(client *rpc2.Client, pwd string) {
	doRpcCall(client, "wallet.NewAddress", []string{pwd})
}

func status(client *rpc2.Client) {
	doRpcCall(client, "wallet.Status", nil)
}

func Unlock(client *rpc2.Client, param []string) {
	doRpcCall(client, "wallet.Unlock", param)
}

func Lock(client *rpc2.Client, param []string) {
	doRpcCall(client, "wallet.Lock", param)
}

func SignData(client *rpc2.Client, param []string) {
	doRpcCall(client, "wallet.SignData", param)
}

func SignDataWithPassphrase(client *rpc2.Client, param []string) {
	doRpcCall(client, "wallet.SignDataWithPassphrase", param)
}

func ImportPriv(client *rpc2.Client, param []string) {
	doRpcCall(client, "wallet.ImportPriv", param)
}

func ExportPriv(client *rpc2.Client, param []string) {
	doRpcCall(client, "wallet.ExportPriv", param)
}

func doRpcCall(client *rpc2.Client, method string, param []string) {
	var s string
	err := client.Call(method, param, &s)
	if err != nil {
		println(err.Error())
	}
	println(method + "\n " + s)
}
