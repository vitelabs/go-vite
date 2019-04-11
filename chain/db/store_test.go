package chain_db

import (
	"fmt"
	"github.com/vitelabs/go-vite/chain/test_tools"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"math/rand"
	"path"
	"testing"
)

func TestStore(t *testing.T) {
	id, _ := types.BytesToHash(crypto.Hash256([]byte("storeTest")))

	store, err := NewStore(path.Join(test_tools.DefaultDataDir(), "test_store"), id)
	if err != nil {
		t.Fatal(err)
	}
	writeBlock(store, 1)
	queryBlock(store, 1)

	// mock flush
	store.Prepare()

	// redo log
	store.RedoLog()

	// commit
	store.Commit()

	writeBlock(store, 2)
	queryBlock(store, 2)

	snapshot(store, []int{1})

	queryBlock(store, 1)
	queryBlock(store, 2)

	writeBlock(store, 3)
	queryBlock(store, 3)

	writeBlock(store, 4)
	queryBlock(store, 4)

	snapshot(store, []int{3})

	queryBlock(store, 1)
	queryBlock(store, 2)
	queryBlock(store, 3)
	queryBlock(store, 4)

}

func writeBlock(store *Store, blockIndex int) {
	batch := store.NewBatch()
	baseNum := uint64(blockIndex * 3)
	batch.Put(chain_utils.Uint64ToBytes(1+baseNum), chain_utils.Uint64ToBytes(1+baseNum))
	batch.Put(chain_utils.Uint64ToBytes(2+baseNum), chain_utils.Uint64ToBytes(2+baseNum))
	batch.Put(chain_utils.Uint64ToBytes(3+baseNum), chain_utils.Uint64ToBytes(3+baseNum))

	randNum := rand.Intn(3) + 1
	for i := 1; i < randNum; i++ {
		batch.Delete(chain_utils.Uint64ToBytes(uint64(i) + baseNum))
	}

	blockHash, _ := types.BytesToHash(crypto.Hash256([]byte(fmt.Sprintf("blockHash%d", blockIndex))))
	store.WriteAccountBlockByHash(batch, blockHash)
}

func snapshot(store *Store, blockIndexList []int) {
	batch := store.NewBatch()
	blockHashList := make([]types.Hash, len(blockIndexList))
	for _, blockIndex := range blockIndexList {
		blockHash, _ := types.BytesToHash(crypto.Hash256([]byte(fmt.Sprintf("blockHash%d", blockIndex))))

		blockHashList = append(blockHashList, blockHash)
	}

	store.WriteSnapshotByHash(batch, blockHashList)
}

func queryBlock(store *Store, blockIndex int) {
	baseNum := uint64(blockIndex * 3)

	fmt.Println("blockIndex", blockIndex)
	fmt.Println(store.Get(chain_utils.Uint64ToBytes(1 + baseNum)))
	fmt.Println(store.Get(chain_utils.Uint64ToBytes(2 + baseNum)))
	fmt.Println(store.Get(chain_utils.Uint64ToBytes(3 + baseNum)))

}
