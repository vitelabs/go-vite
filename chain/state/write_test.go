package chain_state

import (
	"bytes"
	"fmt"
	chain_db "github.com/vitelabs/go-vite/chain/db"
	leveldb "github.com/vitelabs/go-vite/common/db/xleveldb"
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/db/xleveldb/storage"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
	"testing"
	"time"
)

func TestStateDB_Write(t *testing.T) {
	address, err := types.HexToAddress("vite_d789431f1d820506c83fd539a0ae9863d6961382f67341a8b5")

	if err != nil {
		t.Fatal(fmt.Sprintf("address vite_d789431f1d820506c83fd539a0ae9863d6961382f67341a8b5 is not illegal. Error: %s", err.Error()))
	}

	stateDb, err := newTestStateDb(&config.Chain{
		VmLogWhiteList: []types.Address{address},
	})

	if err != nil {
		t.Fatal(err)
	}

	// mock vm log list
	mockHash, err := types.HexToHash("381ab7d6f980d1a732dec29433e59a433e6e23ed0bb6647e67e5f4d0a1e6d0e3")
	if err != nil {
		t.Fatal(err)
	}
	vmLogList := []*ledger.VmLog{{
		Topics: []types.Hash{mockHash},
		Data:   []byte("test vm log list"),
	}}

	// case 1
	vmAccountBlock1 := createAccountBlock(address, vmLogList)
	stateDb.Write(vmAccountBlock1)

	queryVmLogList, err := stateDb.GetVmLogList(vmAccountBlock1.AccountBlock.LogHash)
	if err != nil {
		t.Fatal(err)
	}

	if err := compareVmLogList(vmAccountBlock1.VmDb.GetLogList(), queryVmLogList); err != nil {
		t.Fatal(err)
	}

	// case 2
	vmLogList2 := []*ledger.VmLog{{
		Topics: []types.Hash{mockHash},
		Data:   []byte("test vm log list 222"),
	}}

	noWhiteListAddress := types.AddressPledge
	if err != nil {
		t.Fatal(err)
	}

	vmAccountBlock2 := createAccountBlock(noWhiteListAddress, vmLogList2)

	stateDb.Write(vmAccountBlock2)

	queryVmLogList2, err := stateDb.GetVmLogList(vmAccountBlock2.AccountBlock.LogHash)
	if err != nil {
		t.Fatal(err)
	}

	if err := compareVmLogList(nil, queryVmLogList2); err != nil {
		t.Fatal(err)
	}

	// close state db
	if err := stateDb.Close(); err != nil {
		t.Fatal(err)
	}

	// case 3
	stateDbAll, err := newTestStateDb(&config.Chain{
		VmLogWhiteList: []types.Address{address},
		VmLogAll:       true,
	})

	if err != nil {
		t.Fatal(err)
	}

	stateDbAll.Write(vmAccountBlock1)
	stateDbAll.Write(vmAccountBlock2)

	queryVmLogList3, err := stateDbAll.GetVmLogList(vmAccountBlock1.AccountBlock.LogHash)

	if err != nil {
		t.Fatal(err)
	}

	if err := compareVmLogList(vmAccountBlock1.VmDb.GetLogList(), queryVmLogList3); err != nil {
		t.Fatal(err)
	}

	queryVmLogList4, err := stateDbAll.GetVmLogList(vmAccountBlock2.AccountBlock.LogHash)

	if err != nil {
		t.Fatal(err)
	}
	if err := compareVmLogList(vmAccountBlock2.VmDb.GetLogList(), queryVmLogList4); err != nil {
		t.Fatal(err)
	}

	// close state all db
	if err := stateDbAll.Close(); err != nil {
		t.Fatal(err)
	}

}

func newTestStateDb(cfg *config.Chain) (*StateDB, error) {
	store, err := newMemStore("test_stateDB")
	if err != nil {
		return nil, err
	}

	redoStore, err := newMemStore("test_stateDB_redo")
	if err != nil {
		return nil, err
	}

	return NewStateDBWithStore(newMockChain(), cfg, store, redoStore)

}

func compareVmLogList(correctVmLogList ledger.VmLogList, queryVmLogList ledger.VmLogList) error {
	// check length
	if len(queryVmLogList) != len(correctVmLogList) {
		return errors.New("len(queryVmLogList) != len(correctVmLogList)")
	}

	if len(queryVmLogList) <= 0 {
		return nil
	}

	correctLogHash := correctVmLogList.Hash()

	// check hash
	queryHash := queryVmLogList.Hash()
	if *correctLogHash != *queryHash {
		return errors.New(fmt.Sprintf("*correctLogHash(%s) != *queryHash(%s))", *correctLogHash, *queryHash))
	}

	// check content
	for index, queryVmLog := range queryVmLogList {
		correctVmLog := correctVmLogList[index]
		if !bytes.Equal(queryVmLog.Data, correctVmLog.Data) {
			return errors.New("!bytes.Equal(queryVmLog.Data, correctVmLog.Data)")
		}

		for topicIndex, queryTopic := range queryVmLog.Topics {
			correctTopic := correctVmLog.Topics[topicIndex]
			if queryTopic != correctTopic {
				return errors.New("queryTopic != correctTopic")
			}
		}
	}

	return nil
}

func createAccountBlock(address types.Address, vmLogList ledger.VmLogList) *vm_db.VmAccountBlock {
	vmDB := vm_db.NewEmptyVmDB(&address)
	for _, vmLog := range vmLogList {
		vmDB.AddLog(vmLog)
	}

	vmDB.Finish()

	ab := &ledger.AccountBlock{
		BlockType:      ledger.BlockTypeReceive,
		AccountAddress: address,
		Height:         10,
		LogHash:        vmDB.GetLogListHash(),
	}

	ab.Hash = ab.ComputeHash()

	return &vm_db.VmAccountBlock{
		AccountBlock: ab,
		VmDb:         vmDB,
	}
}

func newMemStore(name string) (*chain_db.Store, error) {
	memLevelDB, err := leveldb.Open(storage.NewMemStorage(), nil)
	if err != nil {
		return nil, err
	}

	store, err := chain_db.NewStoreWithDb("", name, memLevelDB)
	if err != nil {
		return nil, err

	}

	return store, nil
}

type MockChain struct {
}

func (*MockChain) QueryLatestSnapshotBlock() (*ledger.SnapshotBlock, error) {
	now := time.Now()

	sb := &ledger.SnapshotBlock{
		Height:    101,
		Timestamp: &now,
	}

	sb.Hash = sb.ComputeHash()
	return sb, nil
}

func (*MockChain) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	now := time.Now()
	sb := &ledger.SnapshotBlock{
		Height:    100,
		Timestamp: &now,
	}

	sb.Hash = sb.ComputeHash()
	return sb
}

func (*MockChain) GetSnapshotHeightByHash(hash types.Hash) (uint64, error) {
	panic("implement me")
}

func (*MockChain) GetUnconfirmedBlocks(addr types.Address) []*ledger.AccountBlock {
	panic("implement me")
}

func (*MockChain) GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error) {
	panic("implement me")
}

func newMockChain() Chain {
	return &MockChain{}
}
