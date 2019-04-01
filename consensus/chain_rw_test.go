package consensus

import (
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"

	"github.com/golang/mock/gomock"
	"github.com/vitelabs/go-vite/log15"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/config/gen"

	"github.com/vitelabs/go-vite/chain"
)

type mock_ch struct {
}

func (self *mock_ch) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	panic("implement me")
}

func (self *mock_ch) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	panic("implement me")
}

func GetConsensusGroupList() ([]*types.ConsensusGroupInfo, error) {
	info := &types.ConsensusGroupInfo{
		Gid:                    types.SNAPSHOT_GID,
		NodeCount:              10,
		Interval:               1,
		PerCount:               3,
		RandCount:              2,
		RandRank:               100,
		Repeat:                 1,
		CheckLevel:             0,
		CountingTokenId:        types.TokenTypeId{},
		RegisterConditionId:    0,
		RegisterConditionParam: nil,
		VoteConditionId:        0,
		VoteConditionParam:     nil,
		Owner:                  types.Address{},
		PledgeAmount:           nil,
		WithdrawHeight:         0,
	}

	return []*types.ConsensusGroupInfo{info}, nil
}

func (self *mock_ch) GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error) {
	panic("implement me")
}

func (self *mock_ch) GetVoteList(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error) {
	panic("implement me")
}

func (self *mock_ch) GetConfirmedBalanceList(addrList []types.Address, tokenId types.TokenTypeId, sbHash types.Hash) (map[types.Address]*big.Int, error) {
	panic("implement me")
}

func (self *mock_ch) GetSnapshotHeaderBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error) {
	panic("implement me")
}

func (self *mock_ch) GetContractMeta(contractAddress types.Address) (meta *ledger.ContractMeta, err error) {
	panic("implement me")
}

func (self *mock_ch) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	panic("implement me")
}

func (self *mock_ch) GetSnapshotBlockByHash(hash types.Hash) (*ledger.SnapshotBlock, error) {
	panic("implement me")
}

func (self *mock_ch) GetSnapshotHeadersAfterOrEqualTime(endHashHeight *ledger.HashHeight, startTime *time.Time, producer *types.Address) ([]*ledger.SnapshotBlock, error) {
	panic("implement me")
}

func (self *mock_ch) IsGenesisSnapshotBlock(hash types.Hash) bool {
	panic("implement me")
}

func (self *mock_ch) GetRandomSeed(snapshotHash types.Hash, n int) uint64 {
	panic("implement me")
}

func (self *mock_ch) NewDb(dbDir string) (*leveldb.DB, error) {
	panic("implement me")
}

func testDataDir() string {
	return "testdata-consensus"
}

func prepareChain() chain.Chain {
	clearChain(nil)
	c := chain.NewChain(testDataDir(), nil, config_gen.MakeGenesisConfig(""))

	err := c.Init()
	if err != nil {
		panic(err)
	}
	err = c.Start()
	if err != nil {
		panic(err)
	}
	return c
}

func clearChain(c chain.Chain) {
	if c != nil {
		c.Stop()
	}
	err := os.RemoveAll(testDataDir())
	if err != nil {
		panic(err)
	}
}

func Test_chainRw(t *testing.T) {
	c := prepareChain()
	defer clearChain(c)

	//log := log15.New("unittest", "chainrw")
	//rw := newChainRw(c, log)
	//rw.initArray(nil)
}

func NewDb(dirName string) (*leveldb.DB, error) {
	db, err := leveldb.OpenFile(dirName, nil)

	if err != nil {
		return nil, err
	}
	return db, nil
}

func TestChainRw_GetMemberInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	// Assert that Bar() is invoked.
	defer ctrl.Finish()

	dir := "testdata-consensus"
	db, e := NewDb(dir)

	defer func() {
		os.RemoveAll(dir)
	}()
	defer db.Close()

	mch := NewMockChain(ctrl)
	genesisBlock := &ledger.SnapshotBlock{Height: uint64(1), Timestamp: &simpleGenesis}
	genesisBlock.ComputeHash()
	mch.EXPECT().GetLatestSnapshotBlock().Return(genesisBlock).AnyTimes()
	mch.EXPECT().GetGenesisSnapshotBlock().Return(genesisBlock).AnyTimes()
	mch.EXPECT().NewDb(gomock.Any()).Return(db, e).MaxTimes(1)
	infos, err := GetConsensusGroupList()
	mch.EXPECT().GetConsensusGroupList(genesisBlock.Hash).Return(infos, err).MaxTimes(1)

	rw := newChainRw(mch, log15.New("unittest", "chainrw"))
	block := rw.GetLatestSnapshotBlock()
	assert.Equal(t, genesisBlock.Timestamp, block.Timestamp)
	groupInfo, err := rw.GetMemberInfo(types.SNAPSHOT_GID)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	assert.Equal(t, groupInfo.PlanInterval, uint64(30))
}
