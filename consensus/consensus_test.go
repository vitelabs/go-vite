package consensus

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/vitelabs/go-vite/pool/lock"

	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/quota"
)

//
//import (
//	"encoding/json"
//	"strconv"
//	"testing"
//
//	"github.com/vitelabs/go-vite/chain/unittest"
//
//	"time"
//
//	"fmt"
//
//	"path/filepath"
//
//	"math/big"
//
//	"github.com/pkg/errors"
//	"github.com/vitelabs/go-vite/chain"
//	"github.com/vitelabs/go-vite/common"
//	"github.com/vitelabs/go-vite/common/types"
//	"github.com/vitelabs/go-vite/config"
//	"github.com/vitelabs/go-vite/consensus/core"
//	"github.com/vitelabs/go-vite/ledger"
//	"github.com/vitelabs/go-vite/log15"
//
//	"net/http"
//	_ "net/http/pprof"
//)
//
//var log = log15.New("module", "consensusTest")
//
//func TestConsensus(t *testing.T) {
//
//	ch := make(chan string)
//
//	cs := genConsensus(t)
//	cs.Subscribe(types.SNAPSHOT_GID, "snapshot_mock", nil, func(e Event) {
//		ch <- fmt.Sprintf("snapshot: %s, %v", e.Address.String(), e)
//	})
//
//	cs.Subscribe(types.DELEGATE_GID, "contract_mock", nil, func(e Event) {
//		ch <- fmt.Sprintf("account: %s, %v", e.Address.String(), e)
//	})
//
//	for {
//		msg := <-ch
//		log.Info(msg)
//
//	}
//
//}
//
//func TestCommittee_ReadVoteMapByTime(t *testing.T) {
//	cs := genConsensus(t)
//	now := time.Now()
//	u, e := cs.VoteTimeToIndex(types.SNAPSHOT_GID, now)
//	if e != nil {
//		panic(e)
//	}
//
//	details, _, err := cs.ReadVoteMapByTime(types.SNAPSHOT_GID, u)
//	if err != nil {
//		panic(err)
//	}
//	for k, v := range details {
//		t.Log(k, v.Addr, v.Name)
//	}
//}
//
//func TestCommittee_ReadByTime(t *testing.T) {
//	cs := genConsensus(t)
//	now := time.Now()
//	contractR, _, err := cs.ReadByTime(types.DELEGATE_GID, now)
//
//	if err != nil {
//		t.Error(err)
//	}
//	for k, v := range contractR {
//		t.Log(types.DELEGATE_GID, k, v, err)
//	}
//	snapshotR, _, err := cs.ReadByTime(types.SNAPSHOT_GID, now)
//
//	if err != nil {
//		t.Error(err)
//	}
//	for k, v := range snapshotR {
//		t.Log(types.SNAPSHOT_GID, k, v, err)
//	}
//
//	if len(contractR)*3 != len(snapshotR) {
//		t.Error("len error.")
//	}
//
//	contractMap := make(map[types.Address]bool)
//	for _, v := range contractR {
//		contractMap[v.Address] = true
//	}
//
//	for _, v := range snapshotR {
//		if contractMap[v.Address] != true {
//			t.Error("address err", v.Address.String())
//		}
//	}
//}
//
//func genConsensus(t *testing.T) *committee {
//	c := chain.NewChain(&config.Config{DataDir: common.DefaultDataDir()})
//	c.Init()
//	c.Start()
//
//	genesis := chain.GenesisSnapshotBlock
//	cs := NewConsensus(*genesis.Timestamp, c)
//	err := cs.Init()
//	if err != nil {
//		t.Error(err)
//		panic(err)
//	}
//	cs.Start()
//	return cs
//}
//
//func TestChainBlock(t *testing.T) {
//	c := genConsensus(t)
//
//	chn := c.rw.rw.(chain.Chain)
//
//	headHeight := chn.GetLatestSnapshotBlock().Height
//	log.Info("snapshot head height", "height", headHeight)
//
//	for i := uint64(1); i <= headHeight; i++ {
//		block, err := chn.GetSnapshotBlockByHeight(i)
//		if err != nil {
//			t.Error(err)
//		}
//		b, err := c.VerifySnapshotProducer(block)
//		if !b {
//			t.Error("snapshot block verify fail.", "block", block, "err", err)
//		}
//	}
//}
//
//func TestChainRw_checkSnapshotHashValid(t *testing.T) {
//	bc := getChainInstance()
//	rw := chainRw{rw: bc}
//	block, e := bc.GetSnapshotBlockByHeight(129)
//	if e != nil {
//		panic(e)
//	}
//	b2, e := bc.GetSnapshotBlockByHeight(130)
//	if e != nil {
//		panic(e)
//	}
//	err := rw.checkSnapshotHashValid(block.Height, block.Hash, b2.Hash, time.Now())
//	if err != nil {
//		t.Error(err)
//	}
//	err = rw.checkSnapshotHashValid(block.Height, block.Hash, block.Hash, time.Now())
//	if err != nil {
//		t.Error(err)
//	}
//
//	err = rw.checkSnapshotHashValid(b2.Height, b2.Hash, block.Hash, time.Now())
//	t.Log(err)
//	if err == nil {
//		t.Error(err)
//	}
//}
//
//var innerChainInstance chain.Chain
//
//func getChainInstance() chain.Chain {
//	if innerChainInstance == nil {
//
//		innerChainInstance = chain.NewChain(&config.Config{
//
//			DataDir: filepath.Join(common.HomeDir(), "Library/GVite/devdata"),
//			//Chain: &config.Chain{
//			//	KafkaProducers: []*config.KafkaProducer{{
//			//		Topic:      "test003",
//			//		BrokerList: []string{"ckafka-r3rbhht9.ap-guangzhou.ckafka.tencentcloudmq.com:6061"},
//			//	}},
//			//},
//			Chain: &config.Chain{GenesisFile: "/Users/jie/Documents/vite/src/github.com/vitelabs/genesis.json"},
//		})
//		innerChainInstance.Init()
//		innerChainInstance.Start()
//	}
//
//	return innerChainInstance
//}
//
//func getChainInstanceFromPath(path string) chain.Chain {
//	if path == "" {
//		path = "Documents/vite/src/github.com/vitelabs/aaaaaaaa/devdata"
//	}
//	if innerChainInstance == nil {
//		innerChainInstance = chain_unittest.NewChainInstanceFromAbsPath(path, false)
//		innerChainInstance.Start()
//	}
//
//	return innerChainInstance
//}
//
//func TestRead(t *testing.T) {
//	ch := getChainInstance()
//	genesis := chain.GenesisSnapshotBlock
//	info := types.ConsensusGroupInfo{
//		Gid:                    types.SNAPSHOT_GID,
//		NodeCount:              25,
//		Interval:               1,
//		PerCount:               3,
//		RandCount:              1,
//		RandRank:               25,
//		CountingTokenId:        ledger.ViteTokenId,
//		RegisterConditionId:    0,
//		RegisterConditionParam: nil,
//		VoteConditionId:        0,
//		VoteConditionParam:     nil,
//		Owner:                  types.Address{},
//		PledgeAmount:           big.NewInt(0),
//		WithdrawHeight:         0,
//	}
//	reader := core.NewReader(*genesis.Timestamp, &info)
//	periodTime, err := reader.PeriodTime()
//	if err != nil {
//		panic(err)
//	}
//	t.Log(periodTime)
//
//	addr, _ := types.HexToAddress("vite_15acba7c5848c4948b9489255d126ddc434cf7ad0fbbe1c351")
//	regist := &types.Registration{
//		Name:           "wj",
//		NodeAddr:       addr,
//		PledgeAddr:     addr,
//		Amount:         nil,
//		WithdrawHeight: 0,
//		RewardIndex:    0,
//		CancelHeight:   0,
//		HisAddrList:    nil,
//	}
//	block := ch.GetLatestSnapshotBlock()
//	t.Log(*genesis.Timestamp, block.Height, block.Timestamp)
//	index, err := reader.TimeToIndex(*block.Timestamp)
//	if err != nil {
//		panic(err)
//	}
//
//	start := uint64(1)
//	if start+100 < index {
//		start = index - 100
//	}
//	detail, err := reader.VoteDetails(start, index+1, regist, ch)
//	if err != nil {
//		panic(err)
//	}
//
//	t.Logf("%+v", detail)
//
//	t.Logf("PlanNum:%d, ActualNum:%d\n", detail.PlanNum, detail.ActualNum)
//
//	t.Log(len(detail.PeriodM))
//
//	for k, v := range detail.PeriodM {
//		t.Logf("\tkey:%d, len:%d\n", k, len(v.VoteMap))
//		for k2, v2 := range v.VoteMap {
//			t.Logf("\t\tkey:%s \tbalance:%s\n", k2, v2.String())
//		}
//	}
//
//}
//
//func TestRead2(t *testing.T) {
//	ch := getChainInstance()
//	//info := types.ConsensusGroupInfo{
//	//	Gid:                    types.SNAPSHOT_GID,
//	//	NodeCount:              25,
//	//	Interval:               1,
//	//	PerCount:               3,
//	//	RandCount:              1,
//	//	RandRank:               25,
//	//	CountingTokenId:        ledger.ViteTokenId,
//	//	RegisterConditionId:    0,
//	//	RegisterConditionParam: nil,
//	//	VoteConditionId:        0,
//	//	VoteConditionParam:     nil,
//	//	Owner:                  types.Address{},
//	//	PledgeAmount:           big.NewInt(0),
//	//	WithdrawHeight:         0,
//	//}
//	genesis := chain.GenesisSnapshotBlock
//	cs := NewConsensus(*genesis.Timestamp, ch)
//	err := cs.Init()
//	if err != nil {
//		t.Error(err)
//		panic(err)
//	}
//	cs.Start()
//
//	details, height, e := cs.ReadVoteMapByTime(types.SNAPSHOT_GID, 1)
//	for k, v := range details {
//		fmt.Printf("\t%+v, %+v\n", k, v)
//	}
//	fmt.Println()
//
//	fmt.Printf("%d-%s\n", height.Height, height.Hash)
//	fmt.Printf("%+v\n", e)
//}
//
//func TestReader3(t *testing.T) {
//	go func() {
//		t.Log(http.ListenAndServe("localhost:6060", nil))
//	}()
//	ch := getChainInstance()
//	genesis := chain.GenesisSnapshotBlock
//	info := types.ConsensusGroupInfo{
//		Gid:                    types.SNAPSHOT_GID,
//		NodeCount:              25,
//		Interval:               1,
//		PerCount:               3,
//		RandCount:              1,
//		RandRank:               25,
//		CountingTokenId:        ledger.ViteTokenId,
//		RegisterConditionId:    0,
//		RegisterConditionParam: nil,
//		VoteConditionId:        0,
//		VoteConditionParam:     nil,
//		Owner:                  types.Address{},
//		PledgeAmount:           big.NewInt(0),
//		WithdrawHeight:         0,
//	}
//
//	block := ch.GetLatestSnapshotBlock()
//	height := ledger.HashHeight{block.Height, block.Hash}
//	now := time.Now()
//	for i := 0; i < 10000; i++ {
//		//for {
//		core.CalVotes(core.NewGroupInfo(*genesis.Timestamp, info), height, ch)
//		//}
//	}
//	t.Log(time.Now().Sub(now))
//
//	now = time.Now()
//	for i := 0; i < 10000; i++ {
//		ch.GetRegisterList(block.Hash, types.SNAPSHOT_GID)
//	}
//	t.Log("getRegisterList", time.Now().Sub(now))
//
//	now = time.Now()
//	for i := 0; i < 10000; i++ {
//		ch.GetVoteList(block.Hash, info.Gid)
//	}
//	t.Log("getVoteMap", time.Now().Sub(now))
//
//	now = time.Now()
//	for i := 0; i < 10000; i++ {
//		ch.GetBalanceList(block.Hash, ledger.ViteTokenId, []types.Address{})
//	}
//	t.Log("GetBalanceList", time.Now().Sub(now))
//
//	// query vote info
//
//}
//
//func TestCommittee_ReadByIndex(t *testing.T) {
//	ch := getChainInstance()
//	genesis := ch.GetGenesisSnapshotBlock()
//	cs := NewConsensus(*genesis.Timestamp, ch)
//
//	events, voteHeight, _ := cs.ReadByIndex(types.SNAPSHOT_GID, 9550)
//
//	t.Log("vote Height", strconv.FormatUint(voteHeight, 10))
//	for _, v := range events {
//		t.Log(v.Stime.String(), v.Address.String())
//	}
//
//	block := ch.GetLatestSnapshotBlock()
//	infos, err := ch.GetConsensusGroupList(block.Hash)
//	if err != nil {
//		panic(err)
//	}
//	var info *types.ConsensusGroupInfo
//	for _, cs := range infos {
//		if cs.Gid == types.SNAPSHOT_GID {
//			info = cs
//			break
//		}
//	}
//	if info == nil {
//		panic(errors.New("can't find group."))
//	}
//	reader := core.NewReader(*genesis.Timestamp, info)
//	registers, _ := ch.GetRegisterList(block.Hash, types.SNAPSHOT_GID)
//	for _, v := range registers {
//		if v.Name == "s1" {
//			detail, err := reader.VoteDetails(9550, 9550, v, ch)
//			if err != nil {
//				panic(err)
//			}
//			t.Log("planNum", strconv.FormatUint(detail.PlanNum, 10))
//			t.Log("actualNum", strconv.FormatUint(detail.ActualNum, 10))
//			t.Log("address", v.NodeAddr.String())
//		}
//
//	}
//}
//
//func TestCommittee_ReadByIndex2(t *testing.T) {
//
//	gid := types.SNAPSHOT_GID
//	startIndex := uint64(9550)
//	endIndex := uint64(9550)
//	ch := getChainInstance()
//	genesis := ch.GetGenesisSnapshotBlock()
//	cs := NewConsensus(*genesis.Timestamp, ch)
//
//	block := ch.GetLatestSnapshotBlock()
//
//	t.Log(strconv.FormatUint(block.Height, 10))
//
//	registers, err := ch.GetRegisterList(block.Hash, gid)
//	if err != nil {
//		panic(err)
//	}
//	infos, err := ch.GetConsensusGroupList(block.Hash)
//	if err != nil {
//		panic(err)
//	}
//	var info *types.ConsensusGroupInfo
//	for _, cs := range infos {
//		if cs.Gid == gid {
//			info = cs
//			break
//		}
//	}
//	if info == nil {
//		panic(err)
//	}
//	reader := core.NewReader(*genesis.Timestamp, info)
//	u, err := reader.TimeToIndex(*block.Timestamp)
//	if err != nil {
//		panic(err)
//	}
//	if u < endIndex {
//		endIndex = u
//	}
//	if endIndex <= 0 {
//		endIndex = u
//	}
//	ch.GetLatestSnapshotBlock()
//	first, err := ch.GetSnapshotBlockHeadByHeight(3)
//	if err != nil {
//		panic(err)
//	}
//	if first == nil {
//		panic(err)
//	}
//	fromIndex, err := reader.TimeToIndex(*first.Timestamp)
//	if err != nil {
//		panic(err)
//	}
//	if startIndex < fromIndex {
//		startIndex = fromIndex
//	}
//	if startIndex <= 0 {
//		startIndex = fromIndex
//	}
//	type Rate struct {
//		Actual uint64
//		Plan   uint64
//		Rate   uint64
//	}
//	m := make(map[string]interface{})
//
//	for _, register := range registers {
//		detail, err := reader.VoteDetails(startIndex, endIndex, register, ch)
//		if err != nil {
//			panic(err)
//		}
//
//		rate := uint64(0)
//		if detail.PlanNum > 0 {
//			rate = (detail.ActualNum * 10000.0) / detail.PlanNum
//		}
//		m[register.Name] = &Rate{
//			Actual: detail.ActualNum,
//			Plan:   detail.PlanNum,
//			Rate:   rate,
//		}
//	}
//	m["startIndex"] = startIndex
//	m["endIndex"] = endIndex
//	s, _, err := cs.VoteIndexToTime(gid, startIndex)
//	if err != nil {
//		panic(err)
//	}
//	m["startTime"] = s.String()
//	e, _, err := cs.VoteIndexToTime(gid, endIndex)
//	if err != nil {
//		panic(err)
//	}
//	m["endTime"] = e.String()
//
//	bytes, _ := json.Marshal(m)
//	t.Log(string(bytes))
//}
//
//func TestChain(t *testing.T) {
//	ch := getChainInstance()
//
//	addr, _ := types.HexToAddress("vite_000000000000000000000000000000000000000309508ba646")
//	head, _ := ch.GetLatestAccountBlock(&addr)
//
//	for i := head.Height; i > 0; i-- {
//		block, e := ch.GetAccountBlockByHeight(&addr, i)
//		if e != nil {
//			panic(e)
//		}
//		snapshotBlock, e2 := ch.GetSnapshotBlockByHash(&block.SnapshotHash)
//		if e2 != nil {
//			panic(e2)
//		}
//
//		accountBlock, e3 := ch.GetAccountBlockByHash(&block.FromBlockHash)
//		if e3 != nil {
//			panic(e3)
//		}
//
//		s2, err := ch.GetSnapshotBlockByHash(&accountBlock.SnapshotHash)
//		if err != nil {
//			panic(err)
//		}
//
//		t.Log(block.Timestamp.Format("15:04:05"),
//			strconv.FormatUint(i, 10),
//			strconv.FormatUint(snapshotBlock.Height, 10),
//			snapshotBlock.Timestamp.Format("15:04:05"),
//			strconv.FormatUint(s2.Height, 10),
//			s2.Timestamp.Format("15:04:05"),
//			accountBlock.Timestamp.Format("15:04:05"))
//	}
//
//}
//
//func TestChain2(t *testing.T) {
//	ch := getChainInstance()
//
//	head := ch.GetLatestSnapshotBlock()
//
//	for i := head.Height; i > 0; i-- {
//		block, e := ch.GetSnapshotBlockByHeight(i)
//		if e != nil {
//			panic(e)
//		}
//
//		t.Log(block.Timestamp.Format("15:04:05"),
//			strconv.FormatUint(i, 10))
//	}
//
//}
//
//func TestGenCache(t *testing.T) {
//	c := getChainInstanceFromPath("/Users/jie/Library/GVite/testdata")
//
//	cs := NewConsensus(*c.GetGenesisSnapshotBlock().Timestamp, c)
//
//	cs.dbDir = "/Users/jie/Library/GVite/testdata/consensus"
//	cs.Init()
//	cs.dbCache.db.Check()
//	for i := uint64(0); i < 105461; i++ {
//		cs.periods.GetByHeight(i)
//	}
//}
//
//func TestA(t *testing.T) {
//	c := getChainInstanceFromPath("/Users/jie/Library/GVite/testdata")
//
//	cs := NewConsensus(*c.GetGenesisSnapshotBlock().Timestamp, c)
//
//	cs.dbDir = "/Users/jie/Library/GVite/testdata/consensus"
//	cs.Init()
//	head := c.GetLatestSnapshotBlock()
//
//	index, err := cs.VoteTimeToIndex(types.SNAPSHOT_GID, *head.Timestamp)
//	if err != nil {
//		panic(err)
//	}
//
//	fmt.Println(index)
//}
//
//func TestContractProducerVerify(t *testing.T) {
//	c := getChainInstanceFromPath("/Users/jie/Library/GVite/testdata")
//
//	cs := NewConsensus(*c.GetGenesisSnapshotBlock().Timestamp, c)
//
//	cs.dbDir = "/Users/jie/Library/GVite/testdata/consensus"
//	cs.Init()
//	{
//		s := time.Now()
//		i := uint64(105461)
//		p, err := cs.periods.GetByHeight(i)
//		if err != nil {
//			panic(errors.WithMessage(err, fmt.Sprintf("index:%d.", i)))
//		}
//		point := p.(*periodPoint)
//		fmt.Printf("%d, %s, %s, %s, %+v, %+v, %+v\n",
//			point.Height(), time.Now().Sub(s).String(), point.PrevHash(), point.NextHash(), point.proof, point.proof2, point.GetSBPInfos())
//
//	}
//	{
//		head := c.GetLatestSnapshotBlock()
//
//		index, err := cs.VoteTimeToIndex(types.SNAPSHOT_GID, *head.Timestamp)
//		if err != nil {
//			panic(err)
//		}
//		for i := index; i >= 0; i-- {
//			s := time.Now()
//			p, err := cs.periods.GetByHeight(i)
//			if err != nil {
//				panic(errors.WithMessage(err, fmt.Sprintf("index:%d.", i)))
//			}
//			point := p.(*periodPoint)
//			fmt.Printf("%d, %s, %s, %s, %+v\n", point.Height(), time.Now().Sub(s).String(), point.PrevHash(), point.NextHash(), point.GetSBPInfos())
//		}
//	}
//
//}
//

func NewChainInstanceFromDir(dirName string, clear bool, genesis string) (chain.Chain, error) {
	if clear {
		os.RemoveAll(dirName)
	}
	quota.InitQuotaConfig(false, true)
	genesisConfig := &config.Genesis{}
	json.Unmarshal([]byte(genesis), genesisConfig)

	chainInstance := chain.NewChain(dirName, &config.Chain{}, genesisConfig)

	if err := chainInstance.Init(); err != nil {
		return nil, err
	}
	chainInstance.Start()
	return chainInstance, nil
}

func TestConsensus(t *testing.T) {
	//dir := UnitTestDir
	dir := "/Users/jie/Documents/vite/src/github.com/vitelabs/cluster1/ledger_datas/ledger_1/devdata"
	c, err := NewChainInstanceFromDir(dir, false, GenesisJson)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	index := uint64(291471)
	cs := NewConsensus(c, &lock.EasyImpl{})
	cs.Init()
	cs.Start()
	stime, etime, err := cs.VoteIndexToTime(types.SNAPSHOT_GID, index)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	t.Log(stime, etime)

	result, err := cs.SBPReader().(*snapshotCs).ElectionIndex(index)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	for _, v := range result.Plans {
		t.Log(v)
	}

	addresses := types.PubkeyToAddress([]byte{59, 245, 248, 162, 33, 219, 95, 240, 171, 227, 160, 56, 42, 147, 223, 34, 252, 232, 23, 156, 236, 11, 73, 135, 153, 172, 56, 81, 90, 193, 39, 82})

	t.Log(addresses)

}

func TestChainSnapshot(t *testing.T) {
	//dir := UnitTestDir
	dir := "/Users/jie/Documents/vite/src/github.com/vitelabs/cluster1/ledger_datas/ledger_1/devdata"
	c, err := NewChainInstanceFromDir(dir, false, GenesisJson)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	prev := c.GetLatestSnapshotBlock()

	for i := uint64(1); i <= prev.Height; i++ {
		block, err := c.GetSnapshotBlockByHeight(i)
		if err != nil {
			panic(err)
		}

		infos, err := c.GetRegisterList(block.Hash, types.SNAPSHOT_GID)
		if err != nil {
			panic(err)
		}

		vs := ""
		for _, v := range infos {
			vs += fmt.Sprintf("[%s],", v.Name)
		}
		fmt.Printf("height:%d, hash:%s, producer:%s, t:%s, vs:%s\n", block.Height, block.Hash, block.Producer(), block.Timestamp, vs)
		//fmt.Printf("%+v\n", block)
	}

}

func TestChainAcc(t *testing.T) {
	dir := "/Users/jie/Documents/vite/src/github.com/vitelabs/cluster1/ledger_datas/ledger_1/devdata"
	c, err := NewChainInstanceFromDir(dir, false, GenesisJson)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	addr := types.HexToAddressPanic("vite_00000000000000000000000000000000000000042d7ef71894")
	prev, err := c.GetLatestAccountBlock(addr)

	assert.NoError(t, err)
	assert.NotNil(t, prev)
	t.Log(prev)
	return

	for i := uint64(1); i <= prev.Height; i++ {
		block, err := c.GetAccountBlockByHeight(addr, i)
		if err != nil {
			panic(err)
		}
		u, e := c.GetConfirmSnapshotHeaderByAbHash(block.Hash)
		if e != nil {
			panic(e)
		}

		fmt.Printf("height:%d, producer:%s, hash:%s, %s, %d\n", block.Height, block.Producer(), block.Hash, u.Hash, u.Height)
		if i > 3000 {
			break
		}
	}
}

func TestChainAll(t *testing.T) {
	dir := "/Users/jie/Documents/vite/src/github.com/vitelabs/cluster1/ledger_datas/ledger_1/devdata"
	genesisJson := GenesisJson
	c, err := NewChainInstanceFromDir(dir, false, genesisJson)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	//block, e := c.GetSnapshotHeaderByHash(types.HexToHashPanic("35484e694fc9318c3de98311a95b92918a5c4a0d2a392493ee534b82d71923b6"))
	//
	//if e != nil {
	//	t.Error(e)
	//	t.FailNow()
	//}
	//t.Log(block)
	//return

	prev := c.GetLatestSnapshotBlock()
	assert.NotNil(t, prev)

	for i := uint64(1); i <= prev.Height; i++ {
		block, err := c.GetSnapshotBlockByHeight(i)
		if err != nil {
			panic(err)
		}
		accM := make(map[types.Address][]*ledger.AccountBlock)
		for k, v := range block.SnapshotContent {
			for i := v.Height; i > 0; i-- {
				tmpAB, err := c.GetAccountBlockByHeight(k, i)
				assert.NoError(t, err)
				sb, err := c.GetConfirmSnapshotHeaderByAbHash(tmpAB.Hash)
				if sb.Hash == block.Hash {
					accM[k] = append(accM[k], tmpAB)
				} else {
					break
				}
			}
		}

		vs := ""
		vs += fmt.Sprintf("snapshot[%d][%s][%s]\n", block.Height, block.Hash, block.PrevHash)
		for k, v := range accM {
			bs := ""
			detailBs := ""
			for _, b := range v {
				bs += fmt.Sprintf("%d,", b.Height)
				detailBs += fmt.Sprintf("[%d-%s]", b.Height, b.Hash)
			}
			vs += fmt.Sprintf("\taccount[%s][%s][%d]\n", k, bs, block.SnapshotContent[k].Height)
			vs += fmt.Sprintf("\t\tdetails[%s]\n", detailBs)
		}
		fmt.Println(vs)
	}
}

//
//func TestChainRw_GetSeedsBeforeHashH(t *testing.T) {
//	chainInstance := chain.NewChain(&config.Config{
//		DataDir: "/Users/jie/Documents/vite/src/github.com/vitelabs/cluster1/ledger_datas/ledger_bk/devdata",
//		Genesis: chain_unittest.MakeChainConfig("/Users/jie/Documents/vite/src/github.com/vitelabs/cluster1/genesis_test.json"),
//	})
//
//	chainInstance.Init()
//	chainInstance.Start()
//	c := chainInstance
//	//cs := NewConsensus(*c.GetGenesisSnapshotBlock().Timestamp, c)
//
//	hash := types.HexToHashPanic("79ac82df5ce2970d1a36ba25313de1bc9af99a09d3570ac9b5f47abfe9cbb49d")
//
//	block, err := c.GetSnapshotBlockByHash(&hash)
//	if err != nil {
//		panic(err)
//	}
//
//	t.Log(block.Producer())
//
//	m := make(map[types.Address][]*ledger.SnapshotBlock)
//	head := c.GetLatestSnapshotBlock()
//	headHeight := head.Height
//
//	for i := headHeight; i >= types.GenesisHeight; i-- {
//		block, err := c.GetSnapshotBlockByHeight(i)
//		if err != nil {
//			panic(err)
//		}
//		if block.SeedHash == nil {
//			continue
//		}
//		_, ok := m[block.Producer()]
//		//bs, ok := m[block.Producer()]
//		//if len(bs) >= 2 {
//		//	continue
//		//}
//		if ok {
//			m[block.Producer()] = append(m[block.Producer()], block)
//		} else {
//			var arr []*ledger.SnapshotBlock
//			arr = append(arr, block)
//			m[block.Producer()] = arr
//		}
//	}
//
//	for k, v := range m {
//		fmt.Printf("addr:%s, %d\n", k.String(), len(v))
//		var top *ledger.SnapshotBlock
//		for _, v := range v {
//			fmt.Printf("%s, %d, %d, %s\n", v.Hash, v.Height, v.Seed, v.SeedHash)
//			if top != nil {
//				seedHash := ledger.ComputeSeedHash(top.Seed, v.PrevHash, v.Timestamp)
//				fmt.Printf("expected:%s, actual:%s, %t\n", v.SeedHash, seedHash, *v.SeedHash == seedHash)
//			}
//			top = v
//		}
//
//	}
//}
