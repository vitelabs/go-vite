package main

// this is a rpc client for go-vite debug .

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vrpc"
	"github.com/vitelabs/go-vite/vrpc/api"
	"io/ioutil"
	"net/http"
	rpc2 "net/rpc"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"unicode"
)

func main() {

	//fmt.Println("Enter d for Default or any others for Test ")
	//inputReader := bufio.NewReader(os.Stdin)
	//input, err := inputReader.ReadString('\n')
	//dir := common.GoViteTestDataDir()
	//if strings.HasPrefix(input, "d") {
	//	dir = common.DefaultDataDir()
	//}

	ipcapiURL := filepath.Join(common.DefaultDataDir(), vrpc.DefaultIpcFile())
	if runtime.GOOS == "windows" {
		ipcapiURL = vrpc.DefaultIpcFile()
	}
	client, err := vrpc.DialIPC(context.Background(), ipcapiURL)
	if err != nil {
		panic(err)
	}

	help()

	TestStaticApis(client)
	for {
		inputReader := bufio.NewReader(os.Stdin)
		input, err := inputReader.ReadString('\n')
		input = strings.TrimRightFunc(strings.TrimLeftFunc(input, unicode.IsSpace), unicode.IsSpace)

		if err != nil {
			return
		}
		if strings.HasPrefix(input, "quit") {
			return
		}
		if strings.HasPrefix(input, "list") {
			list(client)
		} else if strings.HasPrefix(input, "create") {
			param := strings.Split(strings.TrimRight(input, "\n"), " ")[1:]
			s := "123456"
			if len(param) > 1 {
				s = param[0]
			}
			createAddress(client, s)
		} else if strings.HasPrefix(input, "status") {
			status(client)
		} else if strings.HasPrefix(input, "unlock") {
			param := strings.Split(strings.TrimRight(input, "\n"), " ")[1:]
			Unlock(client, param)
		} else if strings.HasPrefix(input, "SignDataWithPassphrase") {
			param := strings.Split(strings.TrimRight(input, "\n"), " ")[1:]
			SignDataWithPassphrase(client, param)
		} else if strings.HasPrefix(input, "SignData") {
			param := strings.Split(strings.TrimRight(input, "\n"), " ")[1:]
			SignData(client, param)
		} else if strings.HasPrefix(input, "lock") {
			param := strings.Split(strings.TrimRight(input, "\n"), " ")[1:]
			Lock(client, param)
		} else if strings.HasPrefix(input, "importpriv") {
			param := strings.Split(strings.TrimRight(input, "\n"), " ")[1:]
			ImportPriv(client, param)
		} else if strings.HasPrefix(input, "exportpriv") {
			param := strings.Split(strings.TrimRight(input, "\n"), " ")[1:]
			ExportPriv(client, param)
		} else if strings.HasPrefix(input, "peers") {
			PeersCount(client, nil)
		} else if strings.HasPrefix(input, "netinfo") {
			NetworkAvailable(client, nil)
		} else if strings.HasPrefix(input, "getacc") {
			param := strings.Split(strings.TrimRight(input, "\n"), " ")[1:]
			GetAccountByAccAddr(client, param)
		} else if strings.HasPrefix(input, "getblocks") {
			param := strings.Split(strings.TrimRight(input, "\n"), " ")[1:]
			GetBlocksByAccAddr(client, param)
		} else if strings.HasPrefix(input, "txcreate") {
			param := strings.Split(strings.TrimRight(input, "\n"), " ")[1:]
			CreateTxWithPassphrase(client, param)
		} else if strings.HasPrefix(input, "unconfirmblocks") {
			param := strings.Split(strings.TrimRight(input, "\n"), " ")[1:]
			GetUnconfirmedBlocksByAccAddr(client, param)
		} else if strings.HasPrefix(input, "unconfirminfo") {
			param := strings.Split(strings.TrimRight(input, "\n"), " ")[1:]
			GetUnconfirmedInfo(client, param)
		} else if strings.HasPrefix(input, "syncinfo") {
			GetInitSyncInfo(client, nil)
		} else if strings.HasPrefix(input, "newtesttoken") {
			param := strings.Split(strings.TrimRight(input, "\n"), " ")[1:]
			newTesttoken(param)
		} else if strings.HasPrefix(input, "help") {
			help()
		} else {
			fmt.Printf(input)
		}
	}

}

func help() {
	fmt.Println("----------------------- JUST A TEST CLIENT DON'T BE TOO SERIOUS -----------------------------")
	fmt.Println("create [password]:                      create an address by given password(default 123456) ")
	fmt.Println("list:                                   list all address")
	fmt.Println("status:                                 show all address locked or unlocked")
	fmt.Println("unlock [address] [password]:            unlock the address with given passsword(default 123456)")
	fmt.Println("importpriv [hexprivkey] [password]:     import private key and use the given password to generate keystore ")
	fmt.Println("exportpriv [address] [password]:        exprort private key ")
	fmt.Println("peers:                                  show connected peers")
	fmt.Println("netinfo:                                show network info")
	fmt.Println("getacc [address]:                       show account info include balance")
	fmt.Println("getblocks [address] [pageindex]:        show account info include balance")
	fmt.Println("txcreate  [self address] [to address]:  transfer 1 vite you can add password and amout after [to address]")
	fmt.Println("unconfirmblocks [address]:              show unconfirmed blocks in given address ")
	fmt.Println("unconfirminfo [address]:                show unconfirmed info in given address ")
	fmt.Println("syncinfo:                               show first sync info")
	fmt.Println("newtesttoken [address]:                 transfer 100 Vite form Genesis address to given address")
	fmt.Println("help:                                   show help")
	fmt.Println("quit:                                   quit")
}

// wallet
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
	if len(param) == 1 {
		param = append(param, "123456")
	}
	doRpcCall(client, "wallet.UnLock", append(param, []string{"0"}...))
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

// net work
func NetworkAvailable(client *rpc2.Client, param []string) {
	doRpcCall(client, "p2p.NetworkAvailable", param)

}
func PeersCount(client *rpc2.Client, param []string) {
	doRpcCall(client, "p2p.PeersCount", param)
}

// ledger
func CreateTxWithPassphrase(client *rpc2.Client, param []string) {
	if len(param) < 2 {
		println("error params")
		return
	}
	pass := "123456"
	if len(param) >= 3 {
		pass = strings.TrimSpace(param[2])
	}
	amount := "1"
	if len(param) >= 4 {
		amount = param[3]
	}

	tx := api.SendTxParms{
		SelfAddr:    strings.TrimSpace(param[0]),
		ToAddr:      strings.TrimSpace(param[1]),
		Passphrase:  pass,
		TokenTypeId: ledger.MockViteTokenId.String(),
		Amount:      amount,
	}
	doRpcCall(client, "ledger.CreateTxWithPassphrase", tx)
}

func GetBlocksByAccAddr(client *rpc2.Client, param []string) {
	if len(param) == 0 {
		println("err param")
		return
	}
	i := 0
	if len(param) == 2 {
		i, _ = strconv.Atoi(param[1])
	}
	tx := api.GetBlocksParams{
		Addr:  strings.TrimSpace(param[0]),
		Index: i,
		Count: 20,
	}
	doRpcCall(client, "ledger.GetBlocksByAccAddr", tx)
}

func GetUnconfirmedBlocksByAccAddr(client *rpc2.Client, param []string) {
	if len(param) != 2 {
		println("err param")
		return
	}
	i, _ := strconv.Atoi(param[1])
	tx := api.GetBlocksParams{
		Addr:  param[0],
		Index: i,
		Count: 10,
	}
	doRpcCall(client, "ledger.GetUnconfirmedBlocksByAccAddr", tx)
}

func GetAccountByAccAddr(client *rpc2.Client, param []string) {
	doRpcCall(client, "ledger.GetAccountByAccAddr", param)
}

func GetUnconfirmedInfo(client *rpc2.Client, param []string) {
	doRpcCall(client, "ledger.GetUnconfirmedInfo", param)
}

func GetInitSyncInfo(client *rpc2.Client, param []string) {
	doRpcCall(client, "ledger.GetInitSyncInfo", nil)
}

func TestStaticApis(client *rpc2.Client) {
	doRpcCall(client, "common.LogDir", nil)
	doRpcCall(client, "types.IsValidHexTokenTypeId", []string{"asd"})
	doRpcCall(client, "types.IsValidHexAddress", []string{"vite_1cb2ab2738cd913654658e879bef8115eb1aa61a9be9d15c3a"})
	doRpcCall(client, "types.IsValidHexAddress", []string{"vite_1cb2ab2738cd913654658e879bef8115eb1aa61a9be9d15c31"})
}

func doRpcCall(client *rpc2.Client, method string, param interface{}) {
	var s string
	err := client.Call(method, param, &s)
	if err != nil {
		println(err.Error())
	}
	println(method + "\n" + s)
}

type newTokenParams struct {
	AccountAddress string `json:"accountAddress"`
}

//http
func newTesttoken(addr []string) {
	if len(addr) == 0 || !types.IsValidHexAddress(addr[0]) {
		println("address error")
		return
	}
	params := newTokenParams{
		addr[0],
	}
	j, _ := json.Marshal(params)
	println(string(j))

	resp, err := http.Post("https://test.vite.net/api/account/newtesttoken", "application/json", bytes.NewReader(j))
	println("Post")
	if err != nil {
		println(err)
	}
	defer resp.Body.Close()
	all, e := ioutil.ReadAll(resp.Body)
	if e != nil {
		println(e)
	} else {
		println(string(all))
	}

}
