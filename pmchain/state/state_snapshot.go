package chain_state

import (
	"container/list"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pmchain/state/mvdb"
	"github.com/vitelabs/go-vite/pmchain/utils"
	"math/big"
	"sync/atomic"
)

type undoOperationsType struct {
	snapshotHeight uint64

	operations map[uint64]uint64
}

type stateSnapshotManager struct {
	mvDB        *mvdb.MultiVersionDB
	chain       Chain
	cachePolicy map[types.Address]int

	undoOperationsList map[types.Address]*list.List
	pendingUndoLogs    map[types.Address][][]byte
	activeSnapshotMap  map[types.Address]map[uint64]*stateSnapshot

	latestActiveSnapshotId uint64
}

func newStateSnapshotManager(chain Chain, mvDB *mvdb.MultiVersionDB) (*stateSnapshotManager, error) {
	cachePolicy := map[types.Address]int{
		types.AddressRegister: 75 * 5,
		types.AddressVote:     75 * 5,
		types.AddressPledge:   75 * 5,
	}
	cachePolicyLen := len(cachePolicy)

	ssm := &stateSnapshotManager{

		mvDB:        mvDB,
		chain:       chain,
		cachePolicy: cachePolicy,

		undoOperationsList: make(map[types.Address]*list.List, cachePolicyLen),
		pendingUndoLogs:    make(map[types.Address][][]byte, cachePolicyLen),
		activeSnapshotMap:  make(map[types.Address]map[uint64]*stateSnapshot, cachePolicyLen),
	}

	if err := ssm.init(); err != nil {
		return nil, err
	}
	return ssm, nil
}

func (ssm *stateSnapshotManager) init() error {

	latestSnapshotHeight := ssm.chain.GetLatestSnapshotBlock().Height

	for addr, count := range ssm.cachePolicy {

		l := list.New()
		ssm.undoOperationsList[addr] = l
		ssm.activeSnapshotMap[addr] = make(map[uint64]*stateSnapshot)

		latestBlock, err := ssm.chain.GetLatestAccountBlock(&addr)
		if err != nil {
			return err
		}

		if latestBlock == nil {
			continue
		}

		blockHash := &latestBlock.Hash
		currentConfirmedTimes := uint64(0)

		undoOperations := make(map[uint64]uint64)
	LOOP:
		for {
			accountBlockList, err := ssm.chain.GetAccountBlocks(blockHash, 10)
			if err != nil {
				return err
			}

			if len(accountBlockList) <= 0 {
				break
			}

			for _, accountBlock := range accountBlockList {
				confirmedTimes, err := ssm.chain.GetConfirmedTimes(&accountBlock.Hash)
				if err != nil {
					return err
				}
				if currentConfirmedTimes != confirmedTimes {
					l.PushBack(&undoOperationsType{
						operations:     undoOperations,
						snapshotHeight: latestSnapshotHeight - currentConfirmedTimes,
					})
					undoOperations = make(map[uint64]uint64)
					currentConfirmedTimes = confirmedTimes
				}

				if confirmedTimes >= uint64(count) {
					break LOOP
				}

				undoLog, err := ssm.mvDB.GetUndoLog(&accountBlock.Hash)

				if err != nil {
					return err
				}

				ssm.mvDB.ParseUndoLog(undoLog, func(keyId uint64, valueId uint64) {
					undoOperations[keyId] = valueId
				})

			}
			blockHash = &accountBlockList[len(accountBlockList)-1].Hash
		}
	}
	return nil
}

func (ssm *stateSnapshotManager) WriteUndo(block *ledger.AccountBlock, undoLog []byte) {
	ssm.pendingUndoLogs[block.AccountAddress] = append(ssm.pendingUndoLogs[block.AccountAddress], undoLog)
}

func (ssm *stateSnapshotManager) Flush(snapshotHeight uint64) {
	for addr, undoLogList := range ssm.pendingUndoLogs {
		undoOperations := &undoOperationsType{
			snapshotHeight: snapshotHeight,
			operations:     ssm.mvDB.PosSeqUndoLogListToOperations(undoLogList),
		}

		ssm.appendUndoOperations(&addr, undoOperations)

		activeSnapshots := ssm.activeSnapshotMap[addr]
		for _, snapshot := range activeSnapshots {
			snapshot.AddCacheKeyIdValueId(undoOperations.operations)
		}
	}
	ssm.pendingUndoLogs = make(map[types.Address][][]byte, len(ssm.cachePolicy))
}

// TODO too old
func (ssm *stateSnapshotManager) NewStateSnapshot(address *types.Address, snapshotBlockHeight uint64) (interfaces.StateSnapshot, error) {

	keyIdValueIdCache := make(map[uint64]uint64)

	pendingUndoOperations := ssm.mvDB.PosSeqUndoLogListToOperations(ssm.pendingUndoLogs[*address])
	for keyId, valueId := range pendingUndoOperations {
		keyIdValueIdCache[keyId] = valueId
	}

	undoOperationsList := ssm.undoOperationsList[*address]

	current := undoOperationsList.Back()
	for {
		undoOperations := current.Value.(*undoOperationsType)
		if undoOperations.snapshotHeight < snapshotBlockHeight {
			break
		}

		for keyId, valueId := range undoOperations.operations {
			keyIdValueIdCache[keyId] = valueId
		}
		current = current.Prev()
	}

	snapshotId := atomic.AddUint64(&ssm.latestActiveSnapshotId, 1)
	ss, err := newStateSnapshot(ssm, snapshotId, address, ssm.mvDB, ssm.mvDB.LatestKeyId(), keyIdValueIdCache)
	if err != nil {
		return nil, err
	}
	ssm.addActiveStateSnapshot(address, snapshotId, ss)
	return ss, nil
}

func (ssm *stateSnapshotManager) addActiveStateSnapshot(address *types.Address, snapshotId uint64, ss *stateSnapshot) {
	ssm.activeSnapshotMap[*address][snapshotId] = ss
}

func (ssm *stateSnapshotManager) ReleaseActiveStateSnapshot(address *types.Address, snapshotId uint64) {
	ssm.activeSnapshotMap[*address][snapshotId] = nil
}

func (ssm *stateSnapshotManager) getUndoOperations(address *types.Address, count int) map[uint64]uint64 {
	undoOperations := make(map[uint64]uint64)

	undoOperationsList := ssm.undoOperationsList[*address]
	element := undoOperationsList.Back()
	for i := count; i > count+1; i-- {
		currentUndoOperations := element.Value.(map[uint64]uint64)
		for keyId, valueId := range currentUndoOperations {
			undoOperations[keyId] = valueId
		}
		element = element.Prev()
	}

	return undoOperations
}

func (ssm *stateSnapshotManager) appendUndoOperations(address *types.Address, undoOperations *undoOperationsType) {
	l := ssm.undoOperationsList[*address]
	if l.Len() >= ssm.cachePolicy[*address] {
		l.Remove(l.Front())
	}
	l.PushBack(undoOperations)

}

type stateSnapshot struct {
	ssm        *stateSnapshotManager
	snapshotId uint64
	address    *types.Address
	mvDB       *mvdb.MultiVersionDB

	maxKeyId uint64

	cacheKeyIdValueId map[uint64]uint64
}

func newStateSnapshot(ssm *stateSnapshotManager, snapshotId uint64, addr *types.Address,
	mvDB *mvdb.MultiVersionDB, maxKeyId uint64, cacheKeyIdValueId map[uint64]uint64) (*stateSnapshot, error) {
	ss := &stateSnapshot{
		ssm:        ssm,
		snapshotId: snapshotId,
		address:    addr,

		mvDB:              mvDB,
		maxKeyId:          maxKeyId,
		cacheKeyIdValueId: cacheKeyIdValueId,
	}

	return ss, nil
}

func (ss *stateSnapshot) GetBalance(tokenTypeId *types.TokenTypeId) (*big.Int, error) {
	key := chain_utils.CreateBalanceKey(ss.address, tokenTypeId)

	value, err := ss.getValue(key)
	if err != nil {
		return nil, err
	}

	return big.NewInt(0).SetBytes(value), nil
}

func (ss *stateSnapshot) GetValue(key []byte) ([]byte, error) {
	storageKey := chain_utils.CreateStorageKeyPrefix(ss.address, key)

	value, err := ss.getValue(storageKey)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (ss *stateSnapshot) NewStorageIterator(prefix []byte) interfaces.StorageIterator {
	return ss.mvDB.NewSnapshotIterator(prefix, ss.isKeyIdValid, ss.getValueIdFromCache)
}

func (ss *stateSnapshot) Release() {
	ss.ssm.ReleaseActiveStateSnapshot(ss.address, ss.snapshotId)
	ss.mvDB = nil
	ss.cacheKeyIdValueId = nil
}

func (ss *stateSnapshot) AddCacheKeyIdValueId(keyIdValueId map[uint64]uint64) {
	for keyId, valueId := range keyIdValueId {
		if _, ok := ss.cacheKeyIdValueId[keyId]; !ok {
			ss.cacheKeyIdValueId[keyId] = valueId
		}
	}
}

func (ss *stateSnapshot) isKeyIdValid(keyId uint64) bool {
	return keyId <= ss.maxKeyId
}

func (ss *stateSnapshot) getValueIdFromCache(keyId uint64) (uint64, bool) {
	result, ok := ss.cacheKeyIdValueId[keyId]
	return result, ok
}

func (ss *stateSnapshot) getValue(key []byte) ([]byte, error) {
	keyId, err := ss.mvDB.GetKeyId(key)
	if err != nil {
		return nil, err
	}

	valueId, ok := ss.cacheKeyIdValueId[keyId]
	if !ok {
		var err error
		valueId, err = ss.mvDB.GetValueId(keyId)
		if err != nil {
			return nil, err
		}
	}

	return ss.mvDB.GetValueByValueId(valueId)
}
