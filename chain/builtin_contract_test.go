package chain

import (
	"bytes"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"testing"
)

func TestChain_builtInContract(t *testing.T) {
	chainInstance, accounts, _, _, _, snapshotBlockList := SetUp(t, 5, 138, 7)

	testBuiltInContract(t, chainInstance, accounts, snapshotBlockList)
	TearDown(chainInstance)
}

func testBuiltInContract(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {
	t.Run("NewStorageDatabase", func(t *testing.T) {
		NewStorageDatabase(t, chainInstance, accounts, snapshotBlockList)
	})
}

func NewStorageDatabase(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {
	for _, snapshotBlock := range snapshotBlockList {
		for _, account := range accounts {
			sd, err := chainInstance.stateDB.NewStorageDatabase(snapshotBlock.Hash, account.addr)
			if err != nil {
				t.Fatal(err)
			}

			if *sd.Address() != account.addr {
				t.Fatal("error")
			}

			err = checkIterator(account.KeyValue, func() (interfaces.StorageIterator, error) {
				return sd.NewStorageIterator(nil)
			})
			if err != nil {
				t.Fatal(fmt.Sprintf("snapshotBlock: %+v. account: %d, Error: %s", snapshotBlock, account.addr.Bytes(), err.Error()))
			}

			for key, value := range account.SnapshotKeyValue[snapshotBlock.Hash] {
				queryValue, err := sd.GetValue([]byte(key))
				if err != nil {
					t.Fatal("error")
				}
				if !bytes.Equal(value, queryValue) {
					t.Fatal("error")
				}

			}

		}
	}
}
