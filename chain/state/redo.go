package chain_state

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain/db"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/db/xleveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"io"
	"math/big"
	"path"
)

type LogItem struct {
	Storage      [][2][]byte
	BalanceMap   map[types.TokenTypeId]*big.Int
	Code         []byte
	ContractMeta map[types.Address][]byte
	VmLogList    map[types.Hash][]byte
	CallDepth    map[types.Hash]uint16
	Height       uint64
}

type SnapshotLog map[types.Address][]LogItem

func (sl *SnapshotLog) Serialize() ([]byte, error) {
	var valueBuffer bytes.Buffer
	enc := gob.NewEncoder(&valueBuffer)

	err := enc.Encode(sl)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("enc.Encode: %+v. Error: %s", sl, err.Error()))
	}

	return valueBuffer.Bytes(), nil
}

func (sl *SnapshotLog) Deserialize(buf []byte) error {
	var valueBuffer bytes.Buffer
	if _, err := valueBuffer.Write(buf); err != nil {
		return err
	}

	dec := gob.NewDecoder(&valueBuffer)

	if err := dec.Decode(sl); err != nil && err != io.EOF {
		return errors.New(fmt.Sprintf("dec.Decode failed, buffer is %+v. Error: %s", buf, err))
	}
	return nil
}

type FlushingBatch struct {
	Operation byte
	Batch     *leveldb.Batch
}

type Redo struct {
	store *chain_db.Store

	cache *RedoCache

	chain Chain

	retainHeight uint64

	id types.Hash

	flushingBatchMap map[uint64]*FlushingBatch
}

func NewStorageRedo(chain Chain, chainDir string) (*Redo, error) {
	id, _ := types.BytesToHash(crypto.Hash256([]byte("stateDbRedo")))

	var err error
	store, err := chain_db.NewStore(path.Join(chainDir, "state_redo"), id)
	if err != nil {
		return nil, err
	}

	redo := &Redo{
		store:        store,
		chain:        chain,
		cache:        NewRedoCache(),
		retainHeight: 1200,

		id: id,
	}

	redo.initCache()

	store.RegisterAfterRecover(func() {
		redo.initCache()
	})
	return redo, nil
}

// init
func (redo *Redo) initCache() {

	height := uint64(0)

	latestSnapshotBlock, err := redo.chain.QueryLatestSnapshotBlock()
	if err != nil {
		panic(err)
	}
	if latestSnapshotBlock != nil {
		height = latestSnapshotBlock.Height
	}
	redo.cache.Init(height)
}

func (redo *Redo) Close() error {
	if err := redo.store.Close(); err != nil {
		return err
	}
	redo.store = nil
	return nil
}

func (redo *Redo) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock, confirmedBlocks []*ledger.AccountBlock) {

	//nextSnapshotLog := redo.snapshotLogMap[snapshotBlock.Height]
	nextSnapshotLog := redo.cache.Current()
	currentSnapshotLog := make(SnapshotLog, len(nextSnapshotLog))

	if len(nextSnapshotLog) > 0 {
		for j := len(confirmedBlocks) - 1; j >= 0; j-- {
			confirmedBlock := confirmedBlocks[j]
			if _, ok := currentSnapshotLog[confirmedBlock.AccountAddress]; ok {
				continue
			}

			logList, ok := nextSnapshotLog[confirmedBlock.AccountAddress]
			if !ok {
				panic(fmt.Sprintf("InsertSnapshotBlock %d. addr: %s, nextSnapshotLog: %+v, confirmedBlock: %+v\n", snapshotBlock.Height, confirmedBlock.AccountAddress, nextSnapshotLog, confirmedBlock))
			}

			for i := len(logList) - 1; i >= 0; i-- {
				if logList[i].Height <= confirmedBlock.Height {
					currentSnapshotLog[confirmedBlock.AccountAddress] = logList[:i+1]
					if i+1 >= len(logList) {
						delete(nextSnapshotLog, confirmedBlock.AccountAddress)
					} else {
						nextSnapshotLog[confirmedBlock.AccountAddress] = logList[i+1:]

					}
					break
				}
			}

		}

	}

	// write store
	batch := redo.store.NewBatch()

	value, err := currentSnapshotLog.Serialize()
	if err != nil {
		panic(err)
	}

	batch.Put(chain_utils.CreateRedoSnapshot(snapshotBlock.Height), value)

	// rollback stale data
	if snapshotBlock.Height > redo.retainHeight {
		batch.Delete(chain_utils.CreateRedoSnapshot(snapshotBlock.Height - redo.retainHeight))
	}

	redo.store.WriteDirectly(batch)

	// set current
	redo.cache.Set(snapshotBlock.Height, currentSnapshotLog)
	// set next snapshot
	redo.SetCurrentSnapshot(snapshotBlock.Height+1, nextSnapshotLog)

}

func (redo *Redo) HasRedo(snapshotHeight uint64) (bool, error) {
	if _, ok := redo.cache.Get(snapshotHeight); ok {
		return true, nil
	}

	ok, err := redo.store.Has(chain_utils.CreateRedoSnapshot(snapshotHeight))
	if err != nil {
		return false, err
	}
	if !ok {
		ok, err = redo.store.Has(chain_utils.CreateRedoSnapshot(snapshotHeight - 1))
		if err != nil {
			return false, err
		}
	}

	return ok, nil
}

func (redo *Redo) QueryLog(snapshotHeight uint64) (SnapshotLog, bool, error) {
	if snapshotLog, ok := redo.cache.Get(snapshotHeight); ok {
		return snapshotLog, ok, nil
	}

	snapshotLog := make(SnapshotLog)

	value, err := redo.store.GetOriginal(chain_utils.CreateRedoSnapshot(snapshotHeight))
	if err != nil {
		if err != leveldb.ErrNotFound {
			return nil, false, err
		}

		ok, err2 := redo.store.Has(chain_utils.CreateRedoSnapshot(snapshotHeight - 1))
		return snapshotLog, ok, err2
	}

	if len(value) >= 0 {
		if err := snapshotLog.Deserialize(value); err != nil {
			return nil, true, errors.New(fmt.Sprintf("dec.Decode failed, value is %+v. Error: %s", value, err))
		}
	}

	return snapshotLog, true, nil
}

func (redo *Redo) SetCurrentSnapshot(snapshotHeight uint64, logMap SnapshotLog) {
	redo.cache.SetCurrent(snapshotHeight, logMap)
}

func (redo *Redo) AddLog(addr types.Address, log LogItem) {
	redo.cache.AddLog(addr, log)

}

func (redo *Redo) Rollback(chunks []*ledger.SnapshotChunk) {
	batch := redo.store.NewBatch()

	for _, chunk := range chunks {
		if chunk != nil && chunk.SnapshotBlock != nil {
			redo.cache.Delete(chunk.SnapshotBlock.Height)
			batch.Delete(chain_utils.CreateRedoSnapshot(chunk.SnapshotBlock.Height))
		}
	}
	redo.store.RollbackSnapshot(batch)
}

func deleteRedoLog(snapshotLog SnapshotLog, addr types.Address, height uint64) []LogItem {
	if len(snapshotLog) <= 0 {
		return nil
	}
	logList, ok := snapshotLog[addr]
	if !ok {
		return nil
	}
	if len(logList) <= 0 {
		return nil
	}
	if height <= logList[0].Height {
		delete(snapshotLog, addr)
		return logList
	}

	if height > logList[len(logList)-1].Height {
		return nil
	}

	var deletedLogList []LogItem
	for i := len(logList) - 1; i >= 0; i-- {
		if logList[i].Height < height {
			deletedLogList = logList[i+1:]
			logList = logList[:i+1]
			break
		}
	}

	if len(logList) <= 0 {
		delete(snapshotLog, addr)
	} else {
		snapshotLog[addr] = logList
	}
	return deletedLogList
}

func parseRedoLog(snapshotLog SnapshotLog) (map[types.Address]map[string][]byte, map[types.Address]map[types.TokenTypeId]*big.Int, error) {
	keySetMap := make(map[types.Address]map[string][]byte, len(snapshotLog))
	tokenSetMap := make(map[types.Address]map[types.TokenTypeId]*big.Int, len(snapshotLog))

	for addr, redLogList := range snapshotLog {

		kvMap := make(map[string][]byte)
		balanceMap := make(map[types.TokenTypeId]*big.Int)

		for _, redoLog := range redLogList {
			for _, kv := range redoLog.Storage {
				kvMap[string(kv[0])] = kv[1]
			}
			for typeTokenId, balance := range redoLog.BalanceMap {
				balanceMap[typeTokenId] = balance
			}
		}

		keySetMap[addr] = kvMap
		tokenSetMap[addr] = balanceMap

	}
	return keySetMap, tokenSetMap, nil
}
