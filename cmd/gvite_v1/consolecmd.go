package main

import (
	"context"
	"fmt"
	"github.com/vitelabs/go-vite/cmd/console"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/rpc"
	"gopkg.in/urfave/cli.v1"
	rpc2 "net/rpc"
	"path/filepath"
	"runtime"
)

var (
	consoleCommand = cli.Command{
		Action: consoleCmd,
		Name:   "console",
		Usage:  "Start a console",
		//Category: "CONSOLE COMMANDS",
		Description: `
		The Gvite console is an interactive shell for the JavaScript runtime environment`,
	}
	attachCommand = cli.Command{
		Action: remoteConsole,
		Name:   "attach",
		Usage:  "Start an interactive console runtime",
		//Category: "CONSOLE COMMANDS",
		Description: `The gvite console is an interactive shell for the JavaScript runtime environment.`,
	}
)

func consoleCmd(ctx *cli.Context) error {
	config := console.Config{
		DataDir: common.DefaultDataDir(),
		DocRoot: common.DefaultDataDir(),
		Client:  nil,
		Preload: nil,
	}

	console, err := console.New(config)
	if err != nil {
		fmt.Printf("Failed to start the JavaScript console: %v", err)
	}
	defer console.Stop(false)

	//doRpcCall(client, "wallet.ListAddress", nil)
	// Otherwise print the welcome screen and enter interactive mode
	console.Welcome()
	console.Interactive()
	return nil
}

func remoteConsole(ctx *cli.Context) error {
	ipcapiURL := filepath.Join(common.DefaultDataDir(), rpc.DefaultIpcFile())
	if runtime.GOOS == "windows" {
		ipcapiURL = rpc.DefaultIpcFile()
	}
	client, err := rpc.DialIPC(context.Background(), ipcapiURL)
	if err != nil {
		panic(err)
	}

	config := console.Config{
		DataDir: common.DefaultDataDir(),
		DocRoot: common.DefaultDataDir(),
		Client:  client,
		Preload: nil,
	}

	console, err := console.New(config)
	if err != nil {
		fmt.Printf("Failed to start the JavaScript console: %v", err)
	}
	defer console.Stop(false)

	//doRpcCall(client, "wallet.ListAddress", nil)
	// Otherwise print the welcome screen and enter interactive mode
	console.Welcome()
	console.Interactive()
	return nil
}

func doRpcCall(client *rpc2.Client, method string, param interface{}) {
	var s string
	err := client.Call(method, param, &s)
	if err != nil {
		println(err.Error())
	}
}
