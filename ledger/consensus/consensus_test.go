package consensus

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/vitelabs/go-vite/v2/common/config"
	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/ledger/pool/lock"
	"github.com/vitelabs/go-vite/v2/ledger/test_tools"
	"github.com/vitelabs/go-vite/v2/log15"
)

func TestContractDposCs_ElectionIndexReader(t *testing.T) {
	c, tempDir := test_tools.NewTestChainInstance(t.Name(), true, config.MockGenesis())
	defer test_tools.ClearChain(c, tempDir)

	rw := newChainRw(c, log15.New(), &lock.EasyImpl{})
	groupInfo, err := rw.GetMemberInfo(types.DELEGATE_GID)
	assert.NoError(t, err)

	cs := newContractDposCs(groupInfo, rw, log15.New())

	proof := newRollbackProof(rw.rw)
	index := cs.Time2Index(time.Now())
	for i := index; i > 0; i-- {
		_, etime := cs.Index2Time(i)
		hashes, err := proof.ProofHash(etime)
		if err != nil {
			t.Error(err)
			assert.FailNow(t, err.Error())
		}
		if rw.rw.IsGenesisSnapshotBlock(hashes) {
			break
		}

		result, err := cs.ElectionIndex(i)
		assert.NoError(t, err)

		vs := "\n"
		for _, v := range result.Plans {
			vs += fmt.Sprintf("\t\t[%s][%s][%s]\n", v.STime, v.ETime, v.Member)
		}
		t.Log(i, len(result.Plans), hashes, result.STime, result.ETime, vs)
	}
}

func TestConsensus(t *testing.T) {
	c, tempDir := test_tools.NewTestChainInstance(t.Name(), true, config.MockGenesis())
	defer test_tools.ClearChain(c, tempDir)

	index := uint64(291471)
	cs := NewConsensus(c, &lock.EasyImpl{})
	cs.Init(nil)
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
	c, tempDir := test_tools.NewTestChainInstance(t.Name(), true, config.MockGenesis())
	defer test_tools.ClearChain(c, tempDir)

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
	c, tempDir := test_tools.NewTestChainInstance(t.Name(), true, config.MockGenesis())
	defer test_tools.ClearChain(c, tempDir)

	addr := types.HexToAddressPanic("vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a")
	prev, err := c.GetLatestAccountBlock(addr)

	assert.NoError(t, err)
	assert.NotNil(t, prev)
	t.Log(prev)
	t.Logf("prev.Height: %d\n", prev.Height)

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
	c, tempDir := test_tools.NewTestChainInstance(t.Name(), true, config.MockGenesis())
	defer test_tools.ClearChain(c, tempDir)

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
