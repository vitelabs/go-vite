package consensus

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/quota"
)

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
	cs := NewConsensus(c)
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
