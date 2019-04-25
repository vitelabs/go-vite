package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/vitelabs/go-vite/pool/lock"

	"github.com/vitelabs/go-vite/ledger"

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
var dir = flag.String("dataDir", "devdata", "data dir, for example: devdata")

func main() {
	flag.Parse()
	bytes, err := ioutil.ReadFile("genesis.json")
	if err != nil {
		panic(err)
	}

	genesisJson := string(bytes)
	index := uint64(*index)
	c := newChain(*dir, genesisJson)

	cs := consensus.NewConsensus(c, &lock.EasyImpl{})
	cs.Init()
	reader := cs.SBPReader().(consensus.DposReader)

	fmt.Println("------proof info-----------")

	t := reader.GenProofTime(index)

	proofBlock, err := c.GetSnapshotHeaderBeforeTime(&t)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d\t%s\t%s\n", index, t, proofBlock.Hash)

	fmt.Println("--------register---------")
	registerM := make(map[types.Address]string)
	proofHash := proofBlock.Hash
	registers, err := c.GetRegisterList(proofHash, types.SNAPSHOT_GID)
	if err != nil {
		panic(err)
	}
	for _, v := range registers {
		for _, vv := range v.HisAddrList {
			registerM[vv] = v.Name
			fmt.Printf("%s\t%s\n", v.Name, vv)
		}
	}

	fmt.Println("------Election result-----------")
	result, err := reader.ElectionIndex(index)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%d\t%s\t%s\n", result.Index, result.STime, result.ETime)
	for k, v := range result.Plans {
		fmt.Printf("%d\t%s\t%s\t%s\t%s\n", k, v.STime, v.ETime, v.Member, registerM[v.Member])
	}

	info := reader.GetInfo().ConsensusGroupInfo
	votes, e := core.CalVotes(info, proofHash, c)
	if e != nil {
		panic(e)
	}

	fmt.Println("--------votes---------")
	for _, v := range votes {
		fmt.Printf("%s,%s,%s\n", v.Name, v.Addr, v.Balance)
	}
	seed := c.GetRandomSeed(proofHash, 25)
	fmt.Printf("----------seed:%d-------------\n", seed)

	fmt.Println("---------success rate--------")
	rates, err := cs.SBPReader().GetSuccessRateByHour(index - 2)
	if err != nil {
		panic(err)
	}

	for k, v := range rates {
		fmt.Printf("%s:%d\n", k, v)
	}

	stime, _ := cs.SBPReader().GetPeriodTimeIndex().Index2Time(index - 2)
	blocks, err := c.GetSnapshotHeadersAfterOrEqualTime(&ledger.HashHeight{Hash: proofBlock.Hash, Height: proofBlock.Height}, &stime, nil)
	for _, v := range blocks {
		fmt.Printf("%s-%d-%s-%s-%s\n", v.Producer(), v.Height, v.Hash, v.PrevHash, v.Timestamp)
	}
}
