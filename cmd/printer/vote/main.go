package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"sort"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/consensus/core"

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

	head := c.GetLatestSnapshotBlock()
	fmt.Printf("%d\t%s\n", head.Height, head.Hash)

	sHash := types.HexToHashPanic(*hash)
	// query register info
	registerList, err := c.GetRegisterList(sHash, types.SNAPSHOT_GID)
	if err != nil {
		panic(err)
	}
	// query vote info
	votes, err := c.GetVoteList(sHash, types.SNAPSHOT_GID)
	if err != nil {
		panic(err)
	}

	var registers []*consensus.VoteDetails

	// cal candidate
	for _, v := range registerList {
		registers = append(registers, genVoteDetails(c, sHash, v, votes, ledger.ViteTokenId))
	}
	sort.Sort(consensus.ByBalance(registers))

	for k, v := range registers {
		fmt.Sprintf("%d\t%s\t%s\n", k, v.Name, v.CurrentAddr)
		for kk, vv := range v.Addr {
			fmt.Sprintf("\t%s\t%s\n", kk, vv.String())
		}
	}
}

func genVoteDetails(rw chain.Chain, snapshotHash types.Hash, registration *types.Registration, infos []*types.VoteInfo, id types.TokenTypeId) *consensus.VoteDetails {
	var addrs []types.Address
	for _, v := range infos {
		if v.NodeName == registration.Name {
			addrs = append(addrs, v.VoterAddr)
		}
	}
	balanceMap, _ := rw.GetConfirmedBalanceList(addrs, id, snapshotHash)
	balanceTotal := big.NewInt(0)
	for _, v := range balanceMap {
		balanceTotal.Add(balanceTotal, v)
	}
	return &consensus.VoteDetails{
		Vote: core.Vote{
			Name:    registration.Name,
			Addr:    registration.NodeAddr,
			Balance: balanceTotal,
		},
		CurrentAddr:  registration.NodeAddr,
		RegisterList: registration.HisAddrList,
		Addr:         balanceMap,
	}
}
