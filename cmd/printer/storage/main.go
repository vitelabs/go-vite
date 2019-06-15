package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/vm/quota"
)

func newChain(dirName string, genesis string) chain.Chain {
	quota.InitQuotaConfig(false, true)
	genesisConfig := &config.Genesis{}
	json.Unmarshal([]byte(genesis), genesisConfig)

	chainInstance := chain.NewChain(dirName, &config.Chain{}, genesisConfig)

	if err := chainInstance.Init(); err != nil {
		panic(err)
	}
	chainInstance.Start()
	return chainInstance
}

var hash = flag.String("hash", "", "hash")
var dir = flag.String("dataDir", "devdata", "data dir, for example: devdata")

func main() {
	flag.Parse()
	bytes, err := ioutil.ReadFile("genesis.json")
	if err != nil {
		panic(err)
	}

	genesisJson := string(bytes)
	c := newChain(*dir, genesisJson)

	block := c.GetLatestSnapshotBlock()
	fmt.Printf("%d\t%s\n", block.Height, block.Hash)

	_, _, state := c.DBs()

	sHash := types.HexToHashPanic(*hash)

	db, err := state.NewStorageDatabase(sHash, types.AddressConsensusGroup)
	if err != nil {
		panic(err)
	}
	iterator, err := db.NewStorageIterator(nil)
	if err != nil {
		panic(err)
	}

	for iterator.Next() {
		fmt.Println(hex.EncodeToString(iterator.Key()), hex.EncodeToString(iterator.Value()))
	}
}
