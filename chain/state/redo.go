package chain_state

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"io"
	"math/big"
	"path"
	"sync"
)

const (
	optWrite    = 1
	optRollback = 2
	optCover    = 3
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

type SnapshotLog struct {
	RedoLogMap map[types.Address][]LogItem
	FlushOpt   byte
}

//map[types.Address][]LogItem
type FlushingBatch struct {
	Operation byte
	Batch     *leveldb.Batch
}
type Redo struct {
	store *bolt.DB

	chain          Chain
	snapshotLogMap map[uint64]*SnapshotLog

	currentSnapshotHeight uint64

	retainHeight uint64

	id         types.Hash
	snapsMapMu sync.RWMutex

	flushingBatchMap map[uint64]*FlushingBatch
}

func NewStorageRedo(chain Chain, chainDir string) (*Redo, error) {
	id, _ := types.BytesToHash(crypto.Hash256([]byte("stateDbRedo")))

	store, err := bolt.Open(path.Join(chainDir, "kv_redo"), 0600, nil)
	if err != nil {
		return nil, err
	}
	redo := &Redo{
		chain:          chain,
		store:          store,
		snapshotLogMap: make(map[uint64]*SnapshotLog),
		retainHeight:   1200,
		id:             id,
	}

	if err := redo.InitFlushingBatchMap(); err != nil {
		return nil, err
	}
	return redo, nil
}

func (redo *Redo) InitFlushingBatchMap() error {
	height := uint64(0)

	latestSnapshotBlock, err := redo.chain.QueryLatestSnapshotBlock()
	if err != nil {
		return err
	}
	if latestSnapshotBlock != nil {
		height = latestSnapshotBlock.Height
	}

	if err := redo.ResetSnapshot(height + 1); err != nil {
		return err
	}
	return nil
}

func (redo *Redo) Close() error {
	if err := redo.store.Close(); err != nil {
		return err
	}
	redo.store = nil
	return nil
}

func (redo *Redo) ResetSnapshot(snapshotHeight uint64) error {
	redoLogMap, _, err := redo.QueryLog(snapshotHeight)

	if err != nil {
		return err
	}

	redo.SetSnapshot(snapshotHeight, redoLogMap)

	return nil
}

func (redo *Redo) NextSnapshot(nextSnapshotHeight uint64, confirmedBlocks []*ledger.AccountBlock) {

	redoLogMap := redo.snapshotLogMap[redo.currentSnapshotHeight].RedoLogMap
	currentRedoLogMap := make(map[types.Address][]LogItem, len(redoLogMap))

	if len(redoLogMap) > 0 {
		for j := len(confirmedBlocks) - 1; j >= 0; j-- {
			confirmedBlock := confirmedBlocks[j]
			if _, ok := currentRedoLogMap[confirmedBlock.AccountAddress]; ok {
				continue
			}

			logList, ok := redoLogMap[confirmedBlock.AccountAddress]
			if !ok {
				panic(fmt.Sprintf("NextSnapshot %d. addr: %s, redoLogMap: %+v, confirmedBlock: %+v\n", nextSnapshotHeight, confirmedBlock.AccountAddress, redoLogMap, confirmedBlock))
			}

			for i := len(logList) - 1; i >= 0; i-- {
				if logList[i].Height <= confirmedBlock.Height {
					currentRedoLogMap[confirmedBlock.AccountAddress] = logList[:i+1]
					if i+1 >= len(logList) {
						delete(redoLogMap, confirmedBlock.AccountAddress)
					} else {
						redoLogMap[confirmedBlock.AccountAddress] = logList[i+1:]

					}
					break
				}

			}

		}

	}
	// FOR DEBUG
	//fmt.Printf("NextSnapshot_current %d. %+v\n", redo.currentSnapshotHeight, currentRedoLogMap)
	//fmt.Printf("NextSnapshot_netx %d. %+v\n", nextSnapshotHeight, redoLogMap)

	redo.snapshotLogMap[redo.currentSnapshotHeight] = &SnapshotLog{
		RedoLogMap: currentRedoLogMap,
		FlushOpt:   optWrite,
	}

	redo.SetSnapshot(nextSnapshotHeight, redoLogMap)
}

func (redo *Redo) HasRedo(snapshotHeight uint64) (bool, error) {
	if snapshotHeight > redo.currentSnapshotHeight {
		return false, nil
	}

	redo.snapsMapMu.RLock()
	if snapshotLog, ok := redo.snapshotLogMap[snapshotHeight]; ok {
		redo.snapsMapMu.RUnlock()
		return snapshotLog.FlushOpt != optRollback, nil
	}
	redo.snapsMapMu.RUnlock()

	hasRedo := true

	err := redo.store.Update(func(tx *bolt.Tx) error {
		bu := tx.Bucket(chain_utils.Uint64ToBytes(snapshotHeight))
		if bu == nil {
			prevBu := tx.Bucket(chain_utils.Uint64ToBytes(snapshotHeight - 1))
			if prevBu == nil {
				hasRedo = false
			}

		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return hasRedo, nil
}

func (redo *Redo) QueryLog(snapshotHeight uint64) (map[types.Address][]LogItem, bool, error) {
	redo.snapsMapMu.RLock()
	if snapshotLog, ok := redo.snapshotLogMap[snapshotHeight]; ok {
		redo.snapsMapMu.RUnlock()
		return snapshotLog.RedoLogMap, snapshotLog.FlushOpt != optRollback, nil
	}
	redo.snapsMapMu.RUnlock()

	redoLogMap := make(map[types.Address][]LogItem)

	hasRedo := true

	err := redo.store.View(func(tx *bolt.Tx) error {
		bu := tx.Bucket(chain_utils.Uint64ToBytes(snapshotHeight))
		if bu == nil {
			prevBu := tx.Bucket(chain_utils.Uint64ToBytes(snapshotHeight - 1))
			if prevBu == nil {
				hasRedo = false
			}

		} else {
			c := bu.Cursor()

			for k, v := c.First(); k != nil; k, v = c.Next() {
				if len(v) <= 0 {
					continue
				}
				addr, err := types.BytesToAddress(k[:types.AddressSize])
				if err != nil {
					return err
				}

				var valueBuffer bytes.Buffer
				dec := gob.NewDecoder(&valueBuffer)

				if _, err := valueBuffer.Write(v); err != nil {
					return err
				}

				for {
					logItem := &LogItem{}

					if err := dec.Decode(logItem); err != nil {
						if err == io.EOF {
							break
						}
						return errors.New(fmt.Sprintf("dec.Decode failed, value is %+v. Error: %s", v, err))
					}
					redoLogMap[addr] = append(redoLogMap[addr], *logItem)
				}

			}
		}
		return nil
	})

	return redoLogMap, hasRedo, err
}

func (redo *Redo) SetSnapshot(snapshotHeight uint64, logMap map[types.Address][]LogItem) {
	var snapshotLog *SnapshotLog
	snapshotLog, ok := redo.snapshotLogMap[snapshotHeight]
	if ok {
		if snapshotLog.FlushOpt == optRollback {
			snapshotLog.FlushOpt = optCover
		}
	} else {
		snapshotLog = &SnapshotLog{
			FlushOpt: optWrite,
		}
	}

	snapshotLog.RedoLogMap = logMap

	redo.snapshotLogMap[snapshotHeight] = snapshotLog

	redo.currentSnapshotHeight = snapshotHeight
	//fmt.Println("SET SNAPSHOT ", snapshotHeight, logMap)

}

func (redo *Redo) AddLog(addr types.Address, log LogItem) {
	// FOR DEBUG
	redo.snapsMapMu.Lock()
	defer redo.snapsMapMu.Unlock()

	//fmt.Println("ADD LOG ", redo.currentSnapshotHeight, addr, log)

	snapshotLog := redo.snapshotLogMap[redo.currentSnapshotHeight]
	snapshotLog.RedoLogMap[addr] = append(snapshotLog.RedoLogMap[addr], log)

}

func (redo *Redo) Rollback(snapshotHeight uint64) {
	redo.snapshotLogMap[snapshotHeight] = &SnapshotLog{
		FlushOpt: optRollback,
	}
}

func deleteRedoLog(redoLog map[types.Address][]LogItem, addr types.Address, height uint64) []LogItem {
	if len(redoLog) <= 0 {
		return nil
	}
	logList, ok := redoLog[addr]
	if !ok {
		return nil
	}
	if len(logList) <= 0 {
		return nil
	}
	if height <= logList[0].Height {
		delete(redoLog, addr)
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
		delete(redoLog, addr)
	} else {
		redoLog[addr] = logList
	}
	return deletedLogList
}

func parseRedoLog(redoLogMap map[types.Address][]LogItem) (map[types.Address]map[string][]byte, map[types.Address]map[types.TokenTypeId]*big.Int, error) {
	keySetMap := make(map[types.Address]map[string][]byte, len(redoLogMap))
	tokenSetMap := make(map[types.Address]map[types.TokenTypeId]*big.Int, len(redoLogMap))

	for addr, redLogList := range redoLogMap {

		//heightLogs := getSortedHeightLogs(heightLogMap)

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
