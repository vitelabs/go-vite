package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpc/apis"
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
	dir := common.GoViteTestDataDir()
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
		} else if strings.HasPrefix(input, "Peers") {
			PeersCount(client, nil)
		} else if strings.HasPrefix(input, "Net") {
			NetworkAvailable(client, nil)
		} else if strings.HasPrefix(input, "GetAcByAddress") {
			GetAccountByAccAddr(client, nil)
		} else if strings.HasPrefix(input, "TxCreate") {
			param := strings.Split(strings.TrimRight(input, "\n"), " ")[1:]
			CreateTxWithPassphrase(client, param)
		} else if strings.HasPrefix(input, "NowSync") {
			GetInitSyncInfo(client, nil)
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
	doRpcCall(client, "wallet.UnLock", param)
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

func NetworkAvailable(client *rpc2.Client, param []string) {
	doRpcCall(client, "p2p.NetworkAvailable", param)

}
func PeersCount(client *rpc2.Client, param []string) {
	doRpcCall(client, "p2p.PeersCount", param)
}
func GetAccountByAccAddr(client *rpc2.Client, param []string) {
	doRpcCall(client, "ledger.GetAccountByAccAddr", param)
}
func GetInitSyncInfo(client *rpc2.Client, param []string) {
	doRpcCall(client, "ledger.GetInitSyncInfo", nil)
}

func CreateTxWithPassphrase(client *rpc2.Client, param []string) {
	tx := apis.SendTxParms{
		SelfAddr:    param[0],
		ToAddr:      param[1],
		Passphrase:  "123456",
		TokenTypeId: ledger.MockViteTokenId.String(),
		Amount:      "100",
	}
	doRpcCall(client, "ledger.CreateTxWithPassphrase", tx)
}

func doRpcCall(client *rpc2.Client, method string, param interface{}) {
	var s string
	err := client.Call(method, param, &s)
	if err != nil {
		println(err.Error())
	}
	println(method + "\n " + s)
}
