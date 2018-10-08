package main

import (
	"fmt"
	"github.com/vitelabs/go-vite/cmd/utils"
	"github.com/vitelabs/go-vite/log15"
	"gopkg.in/urfave/cli.v1"
)

var (
	initCommand = cli.Command{
		Action:    initGenesis,
		Name:      "init",
		Usage:     "Bootstrap and initialize a new genesis block",
		ArgsUsage: "<genesisPath>",
		Flags: []cli.Flag{
			utils.DataDirFlag,
		},
		//Category: "BLOCKCHAIN COMMANDS",
		Description: `
The init command initializes a new genesis block and definition for the network..

It expects the genesis file as argument.`,
	}

	heightCommand = cli.Command{
		Action:      getSnapshotChainHeight,
		Name:        "height",
		Usage:       "Get snapshot chain height",
		Description: "Get the snapshot chain's height.",
	}
)

func getSnapshotChainHeight(ctx *cli.Context) error {
	mainLog := log15.New("module", "ledgercmd/getSnapshotChainHeight")
	//localconfig := makeConfigNode()
	//vnode, err := vite.New(localconfig)
	//if err != nil {
	//	mainLog.Error(err.Error())
	//}
	//
	//ledgerApi := impl.NewLedgerApi(vnode)
	//var result = new(string)
	//ledgerApi.GetSnapshotChainHeight(nil, result)
	//fmt.Println("getSnapshotChainHeight")
	//fmt.Println(*result)
	return nil
}

func initGenesis(ctx *cli.Context) error {
	fmt.Println("initGensis")
	return nil
}
