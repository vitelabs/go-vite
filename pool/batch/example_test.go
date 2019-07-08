package batch_test

import (
	"fmt"
	"testing"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/pool/batch"
)

// ExampleBatchSnapshot_Batch
func ExampleBatchSnapshot_Batch() {
	b := batch.NewBatch(snapshotExists, accountExists, 1, 100)

	addr := common.MockAddress(1)
	aItem := batch.NewMockSendBlcok(batch.NewGenesisBlock(&addr))

	var accBlocks []batch.Item
	accBlocks = append(accBlocks, aItem)
	sItem := batch.NewMockSnapshotBlock(batch.NewGenesisBlock(nil), accBlocks)

	b.AddAItem(aItem, nil)
	b.AddSItem(sItem)

	err := b.Batch(execute, execute)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func execute(p batch.Batch, l batch.Level, bucket batch.Bucket, version uint64) error {
	fmt.Println("execute ", l.Snapshot(), p.Version(), l.Index(), len(bucket.Items()), version)
	return nil
}

func snapshotExists(hash types.Hash) error {
	return nil
}
func accountExists(hash types.Hash) error {
	return nil
}

func TestExample(t *testing.T) {
	ExampleBatchSnapshot_Batch()
}
