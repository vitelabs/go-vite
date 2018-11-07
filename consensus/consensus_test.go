package consensus

import (
	"testing"

	"time"

	"fmt"

	"path/filepath"

	"math/big"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

var log = log15.New("module", "consensusTest")

func TestConsensus(t *testing.T) {

	ch := make(chan string)

	cs := genConsensus(t)
	cs.Subscribe(types.SNAPSHOT_GID, "snapshot_mock", nil, func(e Event) {
		ch <- fmt.Sprintf("snapshot: %s, %v", e.Address.String(), e)
	})

	cs.Subscribe(types.DELEGATE_GID, "contract_mock", nil, func(e Event) {
		ch <- fmt.Sprintf("account: %s, %v", e.Address.String(), e)
	})

	for {
		msg := <-ch
		log.Info(msg)

	}

}

func TestCommittee_ReadVoteMapByTime(t *testing.T) {
	cs := genConsensus(t)
	now := time.Now()
	u, e := cs.VoteTimeToIndex(types.SNAPSHOT_GID, now)
	if e != nil {
		panic(e)
	}

	details, _, err := cs.ReadVoteMapByTime(types.SNAPSHOT_GID, u)
	if err != nil {
		panic(err)
	}
	for k, v := range details {
		t.Log(k, v.Addr, v.Name)
	}
}

func TestCommittee_ReadByTime(t *testing.T) {
	cs := genConsensus(t)
	now := time.Now()
	contractR, _, err := cs.ReadByTime(types.DELEGATE_GID, now)

	if err != nil {
		t.Error(err)
	}
	for k, v := range contractR {
		t.Log(types.DELEGATE_GID, k, v, err)
	}
	snapshotR, _, err := cs.ReadByTime(types.SNAPSHOT_GID, now)

	if err != nil {
		t.Error(err)
	}
	for k, v := range snapshotR {
		t.Log(types.SNAPSHOT_GID, k, v, err)
	}

	if len(contractR)*3 != len(snapshotR) {
		t.Error("len error.")
	}

	contractMap := make(map[types.Address]bool)
	for _, v := range contractR {
		contractMap[v.Address] = true
	}

	for _, v := range snapshotR {
		if contractMap[v.Address] != true {
			t.Error("address err", v.Address.String())
		}
	}
}

func genConsensus(t *testing.T) *committee {
	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})
	c.Init()
	c.Start()

	genesis := chain.GenesisSnapshotBlock
	cs := NewConsensus(*genesis.Timestamp, c)
	err := cs.Init()
	if err != nil {
		t.Error(err)
		panic(err)
	}
	cs.Start()
	return cs
}

func TestChainBlock(t *testing.T) {
	c := genConsensus(t)

	chn := c.rw.rw.(chain.Chain)

	headHeight := chn.GetLatestSnapshotBlock().Height
	log.Info("snapshot head height", "height", headHeight)

	for i := uint64(1); i <= headHeight; i++ {
		block, err := chn.GetSnapshotBlockByHeight(i)
		if err != nil {
			t.Error(err)
		}
		b, err := c.VerifySnapshotProducer(block)
		if !b {
			t.Error("snapshot block verify fail.", "block", block, "err", err)
		}
	}
}

func TestChainRw_checkSnapshotHashValid(t *testing.T) {
	bc := getChainInstance()
	rw := chainRw{rw: bc}
	block, e := bc.GetSnapshotBlockByHeight(129)
	if e != nil {
		panic(e)
	}
	b2, e := bc.GetSnapshotBlockByHeight(130)
	if e != nil {
		panic(e)
	}
	err := rw.checkSnapshotHashValid(block.Height, block.Hash, b2.Hash)
	if err != nil {
		t.Error(err)
	}
	err = rw.checkSnapshotHashValid(block.Height, block.Hash, block.Hash)
	if err != nil {
		t.Error(err)
	}

	err = rw.checkSnapshotHashValid(b2.Height, b2.Hash, block.Hash)
	t.Log(err)
	if err == nil {
		t.Error(err)
	}
}

var innerChainInstance chain.Chain

func getChainInstance() chain.Chain {
	if innerChainInstance == nil {

		innerChainInstance = chain.NewChain(&config.Config{

			DataDir: filepath.Join(common.HomeDir(), "Library/GVite/devdata/ledger"),
			//Chain: &config.Chain{
			//	KafkaProducers: []*config.KafkaProducer{{
			//		Topic:      "test003",
			//		BrokerList: []string{"ckafka-r3rbhht9.ap-guangzhou.ckafka.tencentcloudmq.com:6061"},
			//	}},
			//},
		})
		innerChainInstance.Init()
		innerChainInstance.Start()
	}

	return innerChainInstance
}

func TestRead(t *testing.T) {
	ch := getChainInstance()
	genesis := chain.GenesisSnapshotBlock
	info := types.ConsensusGroupInfo{
		Gid:                    types.SNAPSHOT_GID,
		NodeCount:              25,
		Interval:               1,
		PerCount:               3,
		RandCount:              1,
		RandRank:               25,
		CountingTokenId:        ledger.ViteTokenId,
		RegisterConditionId:    0,
		RegisterConditionParam: nil,
		VoteConditionId:        0,
		VoteConditionParam:     nil,
		Owner:                  types.Address{},
		PledgeAmount:           big.NewInt(0),
		WithdrawHeight:         0,
	}
	reader := core.NewReader(*genesis.Timestamp, &info)
	periodTime, err := reader.PeriodTime()
	if err != nil {
		panic(err)
	}
	t.Log(periodTime)

	addr, _ := types.HexToAddress("vite_15acba7c5848c4948b9489255d126ddc434cf7ad0fbbe1c351")
	regist := &types.Registration{
		Name:           "wj",
		NodeAddr:       addr,
		PledgeAddr:     addr,
		Amount:         nil,
		WithdrawHeight: 0,
		RewardIndex:    0,
		CancelHeight:   0,
		HisAddrList:    nil,
	}
	block := ch.GetLatestSnapshotBlock()
	t.Log(*genesis.Timestamp, block.Height, block.Timestamp)
	index, err := reader.TimeToIndex(*block.Timestamp)
	if err != nil {
		panic(err)
	}

	start := uint64(1)
	if start+100 < index {
		start = index - 100
	}
	detail, err := reader.VoteDetails(start, index+1, regist, ch)
	if err != nil {
		panic(err)
	}

	t.Logf("%+v", detail)

	t.Logf("PlanNum:%d, ActualNum:%d\n", detail.PlanNum, detail.ActualNum)

	t.Log(len(detail.PeriodM))

	for k, v := range detail.PeriodM {
		t.Logf("\tkey:%d, len:%d\n", k, len(v.VoteMap))
		for k2, v2 := range v.VoteMap {
			t.Logf("\t\tkey:%s \tbalance:%s\n", k2, v2.String())
		}
	}

}

func TestRead2(t *testing.T) {
	ch := getChainInstance()
	//info := types.ConsensusGroupInfo{
	//	Gid:                    types.SNAPSHOT_GID,
	//	NodeCount:              25,
	//	Interval:               1,
	//	PerCount:               3,
	//	RandCount:              1,
	//	RandRank:               25,
	//	CountingTokenId:        ledger.ViteTokenId,
	//	RegisterConditionId:    0,
	//	RegisterConditionParam: nil,
	//	VoteConditionId:        0,
	//	VoteConditionParam:     nil,
	//	Owner:                  types.Address{},
	//	PledgeAmount:           big.NewInt(0),
	//	WithdrawHeight:         0,
	//}
	genesis := chain.GenesisSnapshotBlock
	cs := NewConsensus(*genesis.Timestamp, ch)
	err := cs.Init()
	if err != nil {
		t.Error(err)
		panic(err)
	}
	cs.Start()

	details, height, e := cs.ReadVoteMapByTime(types.SNAPSHOT_GID, 1)
	for k, v := range details {
		fmt.Printf("\t%+v, %+v\n", k, v)
	}
	fmt.Println()

	fmt.Printf("%d-%s\n", height.Height, height.Hash)
	fmt.Printf("%+v\n", e)
}

func TestReader3(t *testing.T) {
	ch := getChainInstance()
	genesis := chain.GenesisSnapshotBlock
	info := types.ConsensusGroupInfo{
		Gid:                    types.SNAPSHOT_GID,
		NodeCount:              25,
		Interval:               1,
		PerCount:               3,
		RandCount:              1,
		RandRank:               25,
		CountingTokenId:        ledger.ViteTokenId,
		RegisterConditionId:    0,
		RegisterConditionParam: nil,
		VoteConditionId:        0,
		VoteConditionParam:     nil,
		Owner:                  types.Address{},
		PledgeAmount:           big.NewInt(0),
		WithdrawHeight:         0,
	}

	block := ch.GetLatestSnapshotBlock()
	height := ledger.HashHeight{block.Height, block.Hash}
	now := time.Now()
	for i := 0; i < 10000; i++ {
		core.CalVotes(core.NewGroupInfo(*genesis.Timestamp, info), height, ch)
	}
	t.Log(time.Now().Sub(now))

	now = time.Now()
	for i := 0; i < 10000; i++ {
		ch.GetRegisterList(block.Hash, types.SNAPSHOT_GID)
	}
	t.Log("getRegisterList", time.Now().Sub(now))

	now = time.Now()
	for i := 0; i < 10000; i++ {
		ch.GetVoteMap(block.Hash, info.Gid)
	}
	t.Log("getVoteMap", time.Now().Sub(now))

	now = time.Now()
	for i := 0; i < 10000; i++ {
		ch.GetBalanceList(block.Hash, ledger.ViteTokenId, []types.Address{})
	}
	t.Log("GetBalanceList", time.Now().Sub(now))

	// query vote info

}
