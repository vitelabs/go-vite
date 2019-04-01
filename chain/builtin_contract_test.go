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
	chainInstance, accounts, _, _, _, snapshotBlockList := SetUp(t, 0, 0, 0)

	testBuiltInContract(t, chainInstance, accounts, snapshotBlockList)
	TearDown(chainInstance)
}

func testBuiltInContract(t *testing.T, chainInstance *chain, accounts map[types.Address]*Account, snapshotBlockList []*ledger.SnapshotBlock) {
	t.Run("NewStorageDatabase", func(t *testing.T) {
		NewStorageDatabase(t, chainInstance, accounts, snapshotBlockList)
	})
	t.Run("GetRegisterList", func(t *testing.T) {
		GetRegisterList(t, chainInstance)
	})
}
func GetRegisterList(t *testing.T, chainInstance *chain) {
	latestSnapshot := chainInstance.GetLatestSnapshotBlock()
	for i := uint64(1); i < latestSnapshot.Height; i++ {
		snapshotBlock, err := chainInstance.GetSnapshotHeaderByHeight(i)
		if err != nil {
			t.Fatal(err)
		}

		registerList, err := chainInstance.GetRegisterList(snapshotBlock.Hash, types.SNAPSHOT_GID)
		if err != nil {
			t.Fatal(err)
		}

		for _, register := range registerList {
			fmt.Println(register)
		}

	}
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
