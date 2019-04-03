package consensus

import (
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/consensus/db"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

func TestPeriodLinkedArray_GetByIndex(t *testing.T) {
	ctrl := gomock.NewController(t)
	// Assert that Bar() is invoked.
	defer ctrl.Finish()

	mch := NewMockChain(ctrl)

	simple := newSimpleCs(log15.New())

	b1 := GenSnapshotBlock(1, "3fc5224e59433bff4f48c83c0eb4edea0e4c42ea697e04cdec717d03e50d5200", types.Hash{}, simpleGenesis)
	b2 := GenSnapshotBlock(2, "3fc5224e59433bff4f48c83c0eb4edea0e4c42ea697e04cdec717d03e50d5200", b1.Hash, simpleGenesis.Add(time.Second))
	b3 := GenSnapshotBlock(3, "3fc5224e59433bff4f48c83c0eb4edea0e4c42ea697e04cdec717d03e50d5200", b2.Hash, simpleGenesis.Add(time.Second*2))
	b4 := GenSnapshotBlock(4, "e0de77ffdc2719eb1d8e89139da9747bd413bfe59781c43fc078bb37d8cbd77a", b3.Hash, simpleGenesis.Add(time.Second*3))
	b5 := GenSnapshotBlock(5, "e0de77ffdc2719eb1d8e89139da9747bd413bfe59781c43fc078bb37d8cbd77a", b4.Hash, simpleGenesis.Add(time.Second*4))
	b6 := GenSnapshotBlock(6, "e0de77ffdc2719eb1d8e89139da9747bd413bfe59781c43fc078bb37d8cbd77a", b5.Hash, simpleGenesis.Add(time.Second*5))

	fmt.Println(b1.Hash)
	fmt.Println(b2.Hash)
	fmt.Println(b6.Hash)

	eTime := simple.GetInfo().GenETime(0)
	mch.EXPECT().GetSnapshotHeaderBeforeTime(gomock.Eq(&eTime)).Return(b6, nil)
	mch.EXPECT().IsGenesisSnapshotBlock(gomock.Not(b1.Hash)).Return(false)
	mch.EXPECT().GetSnapshotBlockByHeight(gomock.Eq(uint64(7))).Return(nil, nil)
	var r []*ledger.SnapshotBlock
	r = append(r, b6)
	r = append(r, b5)
	r = append(r, b4)
	r = append(r, b3)
	r = append(r, b2)
	r = append(r, b1)
	mch.EXPECT().GetSnapshotHeadersAfterOrEqualTime(gomock.Eq(&ledger.HashHeight{Hash: b6.Hash, Height: b6.Height}), gomock.Eq(&simpleGenesis), gomock.Nil()).Return(r, nil)
	periods := newPeriodPointArray(mch, simple)

	point, err := periods.GetByIndex(0)
	assert.NoError(t, err)

	for k, v := range point.Sbps {
		t.Log(k, v)
	}

	assert.Equal(t, uint32(3), point.Sbps[b1.Producer()].FactualNum)
	assert.Equal(t, uint32(3), point.Sbps[b1.Producer()].ExpectedNum)
	assert.Equal(t, uint32(3), point.Sbps[b4.Producer()].FactualNum)
	assert.Equal(t, uint32(3), point.Sbps[b4.Producer()].ExpectedNum)
	assert.Equal(t, &types.Hash{}, point.PrevHash)
	assert.Equal(t, &b6.Hash, point.Hash)
}

func GenSnapshotBlock(height uint64, hexPubKey string, prevHash types.Hash, t time.Time) *ledger.SnapshotBlock {

	pub, err := hex.DecodeString(hexPubKey)
	if err != nil {
		panic(err)
	}

	block := &ledger.SnapshotBlock{
		Hash:            types.Hash{},
		PrevHash:        prevHash,
		Height:          height,
		PublicKey:       pub,
		Signature:       nil,
		Timestamp:       &t,
		Seed:            0,
		SeedHash:        nil,
		SnapshotContent: nil,
	}
	block.Hash = block.ComputeHash()
	return block
}

func prepareConsensusDB() *consensus_db.ConsensusDB {
	clearConsensusDB(nil)
	d, err := leveldb.OpenFile("testdata-consensus", nil)
	if err != nil {
		panic(err)
	}

	db := consensus_db.NewConsensusDB(d)
	return db
}

func clearConsensusDB(db *consensus_db.ConsensusDB) {
	os.RemoveAll("testdata-consensus")
}

func TestHourLinkedArray_GetByIndex(t *testing.T) {
	ctrl := gomock.NewController(t)
	// Assert that Bar() is invoked.
	defer ctrl.Finish()

	//perids := NewMockLinkedArray(ctrl)

}
