package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/consensus/core"
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

var index = flag.Int("index", 0, "consensus index")

func main() {
	flag.Parse()
	bytes, err := ioutil.ReadFile("genesis.json")
	if err != nil {
		panic(err)
	}
	dir := "devdata"
	genesisJson := string(bytes)
	c := newChain(dir, genesisJson)

	cs := consensus.NewConsensus(c)
	cs.Init()
	reader := cs.SBPReader().(consensus.DposReader)

	index := uint64(*index)
	result, err := reader.ElectionIndex(index)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%d\t%s\t%s\n", result.Index, result.STime, result.ETime)
	for k, v := range result.Plans {
		fmt.Printf("%d\t%s\t%s\t%s\t%s\n", k, v.STime, v.ETime, v.Member, v.Name)
	}

	fmt.Println("-----------------")

	t := reader.GenProofTime(index)

	proofBlock, err := c.GetSnapshotHeaderBeforeTime(&t)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d\t%s\t%s\n", index, t, proofBlock.Hash)

	fmt.Println()
	proofHash := proofBlock.Hash
	registers, err := c.GetRegisterList(proofHash, types.SNAPSHOT_GID)
	if err != nil {
		panic(err)
	}
	for _, v := range registers {
		fmt.Printf("%s\t%s\n", v.Name, v.HisAddrList[0])
	}

	info := reader.GetInfo().ConsensusGroupInfo
	votes, e := core.CalVotes(info, proofHash, c)
	if e != nil {
		panic(e)
	}

	fmt.Println("-----------------")
	for _, v := range votes {
		fmt.Printf("%s,%s,%s\n", v.Name, v.Addr, v.Balance)
	}
}
