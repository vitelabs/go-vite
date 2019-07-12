package batch

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/types"
)

func TestBatchSnapshot_AddAItem(t *testing.T) {
	testBatchAndExecute(t, genCase1, verifyCase1)

	testBatchAndExecute(t, genCase2, verifyCase1)

	testBatchAndExecute(t, genCase3, verifyCase3)

	testBatchAndExecute(t, genCase4, verifyCase4)
}

func testBatchAndExecute(t *testing.T, genCaseFn func(chain *mockChain) []*mockItem, verifyFn func(t *testing.T, batch Batch)) {
	// 1. new chain and batch
	chain := newMockChain()
	batch := NewBatch(chain.exists, chain.exists, 1, 100)

	// 2. genCase
	all := genCaseFn(chain)

	// 3. add all item to the batch
	for _, v := range all {
		if v.Owner() != nil {
			err := batch.AddAItem(v, nil)
			if v.expectedErr != nil {
				assert.NotNil(t, err)
			} else {
				assert.NoError(t, err)
			}

			if err != nil {
				break
			}
		} else {
			err := batch.AddSItem(v)
			if v.expectedErr != nil {
				assert.NotNil(t, err)
			} else {
				assert.NoError(t, err)
			}

			if err != nil {
				break
			}
		}
	}
	// 4. verify the batch
	verifyFn(t, batch)

	// 5. execute the batch
	err := batch.Batch(func(p Batch, l Level, bucket Bucket, version uint64) error {
		for _, v := range bucket.Items() {
			t.Log("snapshot insert", fmt.Sprintf("height:%d", v.Height()))
		}
		return chain.execute(p, l, bucket, version)
	}, func(p Batch, l Level, bucket Bucket, version uint64) error {
		for _, v := range bucket.Items() {
			t.Log("account insert", v.Owner(), fmt.Sprintf("height:%d", v.Height()))
		}
		return chain.execute(p, l, bucket, version)
	})

	// 6. assert the result
	assert.NoError(t, err)
	for _, level := range batch.Levels() {
		for _, bu := range level.Buckets() {
			for _, im := range bu.Items() {
				keys, _, _ := im.ReferHashes()
				for _, vv := range keys {
					assert.Nil(t, chain.exists(vv))
				}
			}
		}
	}
}

func initChain(addrs []types.Address, chain *mockChain) (map[types.Address]Item, Item) {
	gensMap := make(map[types.Address]Item)
	for _, e := range addrs {
		tmp := e
		genesis := NewGenesisBlock(&tmp)
		gensMap[tmp] = genesis
		chain.insert(genesis)
	}

	genesisSnapshot := NewGenesisBlock(nil)
	chain.insert(genesisSnapshot)

	return gensMap, genesisSnapshot
}

/**
1. S1,S2
2. AS2,AS1,CS1,BS1,CS2
3. BR2[AS1],BR3[AS2]
4. CR3[BS1],AR3[CS1]
5. S3[AR3,BR3,CR3],S4,S5
6. AS4,BS4,CS4
*/
func genCase4(chain *mockChain) []*mockItem {
	var all []*mockItem

	addrA := common.MockAddress(0)
	addrB := common.MockAddress(1)
	addrC := common.MockAddress(2)

	var addrs []types.Address
	addrs = append(addrs, addrA)
	addrs = append(addrs, addrB)
	addrs = append(addrs, addrC)

	blocks, geneS := initChain(addrs, chain)
	s1 := NewMockSnapshotBlock(geneS, nil)
	s2 := NewMockSnapshotBlock(s1, nil)

	as1 := NewMockSendBlcok(blocks[addrA])
	as2 := NewMockSendBlcok(as1)
	cs1 := NewMockSendBlcok(blocks[addrC])
	bs1 := NewMockSendBlcok(blocks[addrB])
	cs2 := NewMockSendBlcok(cs1)

	br2 := NewMockReceiveBlcok(bs1, as1.Hash())
	br3 := NewMockReceiveBlcok(br2, as2.Hash())

	cr3 := NewMockReceiveBlcok(cs2, bs1.Hash())
	ar3 := NewMockReceiveBlcok(as2, cs1.Hash())

	var accBlocksS3 []Item
	accBlocksS3 = append(accBlocksS3, ar3)
	accBlocksS3 = append(accBlocksS3, br3)
	accBlocksS3 = append(accBlocksS3, cr3)
	s3 := NewMockSnapshotBlock(s2, accBlocksS3)

	s4 := NewMockSnapshotBlock(s3, nil)
	s5 := NewMockSnapshotBlock(s4, nil)

	as4 := NewMockSendBlcok(ar3)
	bs4 := NewMockSendBlcok(br3)
	cs4 := NewMockSendBlcok(cr3)

	all = append(all, s1)
	all = append(all, s2)
	all = append(all, as2)
	as2.expectedErr = errors.New("error!")
	all = append(all, as1)
	all = append(all, cs1)
	all = append(all, bs1)
	all = append(all, cs2)
	all = append(all, br2)
	all = append(all, br3)
	all = append(all, cr3)
	all = append(all, ar3)
	all = append(all, s3)
	all = append(all, s4)
	all = append(all, s5)
	all = append(all, as4)
	all = append(all, bs4)
	all = append(all, cs4)

	return all
}

/**
1. S1,S2
2. AS1,AS2,CS1,BS1,CS2
3. BR2[AS1],BR3[AS2]
4. CR3[BS1],AR3[CS1]
5. S3[AR3,BR3,CR3],S4,S5
6. AS4,BS4,CS4
*/
func genCase3(chain *mockChain) []*mockItem {
	var all []*mockItem

	addrA := common.MockAddress(0)
	addrB := common.MockAddress(1)
	addrC := common.MockAddress(2)

	var addrs []types.Address
	addrs = append(addrs, addrA)
	addrs = append(addrs, addrB)
	addrs = append(addrs, addrC)

	blocks, geneS := initChain(addrs, chain)
	s1 := NewMockSnapshotBlock(geneS, nil)
	s2 := NewMockSnapshotBlock(s1, nil)

	as1 := NewMockSendBlcok(blocks[addrA])
	as2 := NewMockSendBlcok(as1)
	cs1 := NewMockSendBlcok(blocks[addrC])
	bs1 := NewMockSendBlcok(blocks[addrB])
	cs2 := NewMockSendBlcok(cs1)

	br2 := NewMockReceiveBlcok(bs1, as1.Hash())
	br3 := NewMockReceiveBlcok(br2, as2.Hash())

	cr3 := NewMockReceiveBlcok(cs2, bs1.Hash())
	ar3 := NewMockReceiveBlcok(as2, cs1.Hash())

	var accBlocksS3 []Item
	accBlocksS3 = append(accBlocksS3, ar3)
	accBlocksS3 = append(accBlocksS3, br3)
	accBlocksS3 = append(accBlocksS3, cr3)
	s3 := NewMockSnapshotBlock(s2, accBlocksS3)

	s4 := NewMockSnapshotBlock(s3, nil)
	s5 := NewMockSnapshotBlock(s4, nil)

	as4 := NewMockSendBlcok(ar3)
	bs4 := NewMockSendBlcok(br3)
	cs4 := NewMockSendBlcok(cr3)

	all = append(all, s1)
	all = append(all, s2)
	all = append(all, as1)
	all = append(all, as2)
	all = append(all, cs1)
	all = append(all, bs1)
	all = append(all, cs2)
	all = append(all, br2)
	all = append(all, br3)
	all = append(all, cr3)
	all = append(all, ar3)
	all = append(all, s3)
	all = append(all, s4)
	all = append(all, s5)
	all = append(all, as4)
	all = append(all, bs4)
	all = append(all, cs4)
	return all
}

func verifyCase4(t *testing.T, batch Batch) {
	levels := batch.Levels()
	assert.Equal(t, 1, len(levels))

	assert.Equal(t, 2, len(levels[0].Buckets()[0].Items()))
	assert.Equal(t, 2, levels[0].Size())

}

/**
1. AS1
2. BR1[AS1]
3. AS2
4. S1[AS2,BR1]
*/
func genCase2(chain *mockChain) []*mockItem {
	var all []*mockItem

	addrA := common.MockAddress(0)
	addrB := common.MockAddress(1)
	var addrs []types.Address
	addrs = append(addrs, addrA)
	addrs = append(addrs, addrB)

	blocks, geneS := initChain(addrs, chain)
	as1 := NewMockSendBlcok(blocks[addrA])

	br1 := NewMockReceiveBlcok(blocks[addrB], as1.Hash())

	as2 := NewMockSendBlcok(as1)

	var accBlocks []Item
	accBlocks = append(accBlocks, as2)
	accBlocks = append(accBlocks, br1)
	s1 := NewMockSnapshotBlock(geneS, accBlocks)
	all = append(all, as1)
	all = append(all, br1)
	all = append(all, as2)
	all = append(all, s1)
	return all

}

/**
1. AS1, AS2
2. BR1[AS1]
3. S1[AS2,BR1]
*/
func genCase1(chain *mockChain) []*mockItem {
	var all []*mockItem

	addrA := common.MockAddress(0)
	addrB := common.MockAddress(1)
	var addrs []types.Address
	addrs = append(addrs, addrA)
	addrs = append(addrs, addrB)

	blocks, geneS := initChain(addrs, chain)

	as1 := NewMockSendBlcok(blocks[addrA])
	as2 := NewMockSendBlcok(as1)

	br1 := NewMockReceiveBlcok(blocks[addrB], as1.Hash())

	var accBlocks []Item
	accBlocks = append(accBlocks, as2)
	accBlocks = append(accBlocks, br1)
	s1 := NewMockSnapshotBlock(geneS, accBlocks)
	all = append(all, as1)
	all = append(all, as2)
	all = append(all, br1)
	all = append(all, s1)
	return all
}

func verifyCase3(t *testing.T, batch Batch) {
	levels := batch.Levels()
	assert.Equal(t, 5, len(levels))

	assert.Equal(t, 2, len(levels[0].Buckets()[0].Items()))
	assert.Equal(t, 2, levels[0].Size())

	assert.Equal(t, 3, len(levels[1].Buckets()))
	assert.Equal(t, 5, levels[1].Size())

	assert.Equal(t, 3, len(levels[2].Buckets()))
	assert.Equal(t, 4, levels[2].Size())

	assert.Equal(t, 1, len(levels[3].Buckets()))
	assert.Equal(t, 3, levels[3].Size())

	assert.Equal(t, 3, len(levels[4].Buckets()))
	assert.Equal(t, 3, levels[4].Size())

	assert.True(t, levels[0].Snapshot())
	assert.False(t, levels[1].Snapshot())
	assert.False(t, levels[2].Snapshot())
	assert.True(t, levels[3].Snapshot())
	assert.False(t, levels[4].Snapshot())

}

/**
1. AS1, AS2
2. BR1[AS1]
3. S1[AS2,BR1]
*/
func verifyCase1(t *testing.T, batch Batch) {
	levels := batch.Levels()
	assert.Equal(t, 3, len(levels))

	assert.Equal(t, 1, len(levels[0].Buckets()))
	assert.Equal(t, 1, len(levels[1].Buckets()))
	assert.Equal(t, 1, len(levels[2].Buckets()))

	assert.Equal(t, 2, len(levels[0].Buckets()[0].Items()))
	assert.Equal(t, 1, len(levels[1].Buckets()[0].Items()))
	assert.Equal(t, 1, len(levels[2].Buckets()[0].Items()))

	assert.False(t, levels[0].Snapshot())
	assert.False(t, levels[1].Snapshot())
	assert.True(t, levels[2].Snapshot())
}
