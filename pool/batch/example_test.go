package batch

import (
	"fmt"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
)

type mockItem struct {
	prevHash *types.Hash
	hash     types.Hash
	height   uint64
	addr     *types.Address
}

func (m *mockItem) ReferHashes() ([]types.Hash, []types.Hash, *types.Hash) {
	return []types.Hash{m.hash}, nil, m.prevHash
}

func (m *mockItem) Owner() *types.Address {
	return m.addr
}

func (m *mockItem) Hash() types.Hash {
	return m.hash
}

func (m *mockItem) Height() uint64 {
	return m.height
}

func (m *mockItem) PrevHash() types.Hash {
	return *m.prevHash
}

// ExampleBatchSnapshot_Batch
func ExampleBatchSnapshot_Batch() {
	batch := NewBatch(snapshotExists, accountExists, 1, 100)

	prev := common.MockHash(0)
	sItem := &mockItem{prevHash: &prev, hash: common.MockHash(1), height: 1, addr: nil}

	prev = common.MockHash(0)
	addr := common.MockAddress(1)
	aItem := &mockItem{prevHash: &prev, hash: common.MockHash(1), height: 1, addr: &addr}

	batch.AddAItem(aItem, nil)
	batch.AddSItem(sItem)
	err := batch.Batch(execute, execute)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func execute(p Batch, l Level, bucket Bucket, version uint64) error {
	fmt.Println("execute ", p.Version(), l.Index(), len(bucket.Items()), version)
	return nil
}

func snapshotExists(hash types.Hash) error {
	return nil
}
func accountExists(hash types.Hash) error {
	return nil
}
