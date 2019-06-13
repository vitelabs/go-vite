package batch

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common"
)

func TestBatchSnapshot_AddAItem(t *testing.T) {
	testBatchAndExecute(t, genCase1, verifyCase1)
}

func testBatchAndExecute(t *testing.T, genCaseFn func(chain *mockChain) []Item, verifyFn func(t *testing.T, batch Batch)) {
	// 1. new chain and batch
	chain := newMockChain()
	batch := NewBatch(chain.exists, chain.exists, 1, 100)

	// 2. genCase
	all := genCaseFn(chain)

	// 3. add all item to the batch
	for _, v := range all {
		if v.Owner() != nil {
			err := batch.AddAItem(v, nil)
			assert.NoError(t, err)
		} else {
			err := batch.AddSItem(v)
			assert.NoError(t, err)
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
	for _, v := range all {
		keys, _, _ := v.ReferHashes()
		for _, vv := range keys {
			assert.Nil(t, chain.exists(vv))
		}
	}
}

/**
1. AS1, AS2
2. BR1[AS1]
3. S1[AS2,BR1]
*/
func genCase1(chain *mockChain) []Item {
	var all []Item

	addrA := common.MockAddress(1)
	addrB := common.MockAddress(2)

	genesisAddrA := NewGenesisBlock(&addrA)
	genesisAddrB := NewGenesisBlock(&addrB)
	genesisSnapshot := NewGenesisBlock(nil)
	chain.insert(genesisAddrA)
	chain.insert(genesisAddrB)
	chain.insert(genesisSnapshot)

	as1 := NewMockSendBlcok(genesisAddrA)
	as2 := NewMockSendBlcok(as1)

	br1 := NewMockReceiveBlcok(genesisAddrB, as1.Hash())

	var accBlocks []Item
	accBlocks = append(accBlocks, as2)
	accBlocks = append(accBlocks, br1)
	s1 := NewMockSnapshotBlock(genesisSnapshot, accBlocks)
	all = append(all, as1)
	all = append(all, as2)
	all = append(all, br1)
	all = append(all, s1)
	return all
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
