package vm

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/crypto"
	"math/big"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
)

var (
	balanceKey = "$BALANCE"
	codeKey    = "$CODE"
)

func getBalanceKey(tokenId *types.TokenTypeId) string {
	return balanceKey + tokenId.String()
}

func getCodeKey(addr types.Address) string {
	return codeKey + addr.String()
}

// test database for single call
type memoryDatabase struct {
	addr            types.Address
	storage         map[string][]byte
	originalStorage map[string][]byte
	logList         []*ledger.VmLog
	sb              *ledger.SnapshotBlock
}

func NewMemoryDatabase(addr types.Address, sb *ledger.SnapshotBlock) *memoryDatabase {
	return &memoryDatabase{
		addr:            addr,
		storage:         make(map[string][]byte),
		originalStorage: make(map[string][]byte),
		logList:         make([]*ledger.VmLog, 0),
		sb:              sb,
	}
}
func (db *memoryDatabase) GetBalance(addr *types.Address, tokenTypeId *types.TokenTypeId) *big.Int {
	if *addr != db.addr {
		return big.NewInt(0)
	}
	if balance, ok := db.storage[getBalanceKey(tokenTypeId)]; ok {
		return new(big.Int).SetBytes(balance)
	} else {
		return big.NewInt(0)
	}
}
func (db *memoryDatabase) SubBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
	balance := db.GetBalance(&db.addr, tokenTypeId)
	if balance.Cmp(amount) >= 0 {
		db.storage[getBalanceKey(tokenTypeId)] = balance.Sub(balance, amount).Bytes()
	}
}
func (db *memoryDatabase) AddBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
	balance := db.GetBalance(&db.addr, tokenTypeId)
	if balance.Sign() == 0 && amount.Sign() == 0 {
		return
	}
	db.storage[getBalanceKey(tokenTypeId)] = balance.Add(balance, amount).Bytes()
}
func (db *memoryDatabase) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

// forward=true return [startHeight, startHeight+count), forward=false return (startHeight-count, startHeight]
func (db *memoryDatabase) GetSnapshotBlocks(startHeight uint64, count uint64, forward, containSnapshotContent bool) []*ledger.SnapshotBlock {
	blockList := make([]*ledger.SnapshotBlock, 0)
	return blockList
}
func (db *memoryDatabase) GetSnapshotBlockByHash(hash *types.Hash) *ledger.SnapshotBlock {
	return nil
}

func (db *memoryDatabase) GetAccountBlockByHash(hash *types.Hash) *ledger.AccountBlock {
	return nil
}
func (db *memoryDatabase) GetSelfAccountBlockByHeight(height uint64) *ledger.AccountBlock {
	return nil
}
func (db *memoryDatabase) Reset() {}
func (db *memoryDatabase) IsAddressExisted(addr *types.Address) bool {
	return false
}
func (db *memoryDatabase) SetContractGid(gid *types.Gid, addr *types.Address) {
}
func (db *memoryDatabase) SetContractCode(code []byte) {
	db.storage[getCodeKey(db.addr)] = code
}
func (db *memoryDatabase) GetContractCode(addr *types.Address) []byte {
	if code, ok := db.storage[getCodeKey(db.addr)]; ok {
		return code
	}
	return nil
}

func (db *memoryDatabase) GetOriginalStorage(key []byte) []byte {
	if data, ok := db.originalStorage[hex.EncodeToString(key)]; ok {
		return data
	} else {
		return nil
	}
}

func (db *memoryDatabase) GetStorage(addr *types.Address, key []byte) []byte {
	if data, ok := db.storage[hex.EncodeToString(key)]; ok {
		return data
	} else {
		return nil
	}
}
func (db *memoryDatabase) SetStorage(key []byte, value []byte) {
	if len(value) == 0 {
		delete(db.storage, hex.EncodeToString(key))
	} else {
		db.storage[hex.EncodeToString(key)] = value
	}
}
func (db *memoryDatabase) PrintStorage() string {
	str := "["
	for key, value := range db.storage {
		str += key + "=>" + hex.EncodeToString(value) + ", "
	}
	str += "]"
	return str
}
func (db *memoryDatabase) GetStorageHash() *types.Hash {
	return &types.Hash{}
}
func (db *memoryDatabase) AddLog(log *ledger.VmLog) {
	db.logList = append(db.logList, log)
}
func (db *memoryDatabase) GetLogListHash() *types.Hash {
	if len(db.logList) == 0 {
		return nil
	} else {
		var source []byte
		for _, vmLog := range db.logList {
			for _, topic := range vmLog.Topics {
				source = append(source, topic.Bytes()...)
			}
			source = append(source, vmLog.Data...)
		}

		hash, _ := types.BytesToHash(crypto.Hash256(source))
		return &hash
	}
}

func (db *memoryDatabase) NewStorageIterator(addr *types.Address, prefix []byte) vmctxt_interface.StorageIterator {
	return nil
}

func (db *memoryDatabase) CopyAndFreeze() vmctxt_interface.VmDatabase {
	return db
}

func (db *memoryDatabase) GetGid() *types.Gid {
	return nil
}
func (db *memoryDatabase) Address() *types.Address {
	return nil
}
func (db *memoryDatabase) CurrentSnapshotBlock() *ledger.SnapshotBlock {
	return db.sb
}
func (db *memoryDatabase) PrevAccountBlock() *ledger.AccountBlock {
	return nil
}
func (db *memoryDatabase) UnsavedCache() vmctxt_interface.UnsavedCache {
	return nil
}

func (db *memoryDatabase) GetStorageBySnapshotHash(addr *types.Address, key []byte, snapshotHash *types.Hash) []byte {
	return db.GetStorage(addr, key)
}
func (db *memoryDatabase) NewStorageIteratorBySnapshotHash(addr *types.Address, prefix []byte, snapshotHash *types.Hash) vmctxt_interface.StorageIterator {
	return db.NewStorageIterator(addr, prefix)
}

func (db *memoryDatabase) GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error) {
	return abi.GetActiveConsensusGroupList(db, &snapshotHash), nil
}
func (db *memoryDatabase) GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error) {
	return abi.GetCandidateList(db, gid, &snapshotHash), nil
}
func (db *memoryDatabase) GetVoteMap(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error) {
	return abi.GetVoteList(db, gid, &snapshotHash), nil
}
func (db *memoryDatabase) GetBalanceList(snapshotHash types.Hash, tokenTypeId types.TokenTypeId, addressList []types.Address) (map[types.Address]*big.Int, error) {
	balanceList := make(map[types.Address]*big.Int)
	for _, addr := range addressList {
		balanceList[addr] = db.GetBalance(&addr, &tokenTypeId)
	}
	return balanceList, nil
}
func (db *memoryDatabase) GetSnapshotBlockBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (db *memoryDatabase) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return db.CurrentSnapshotBlock()
}

func (db *memoryDatabase) DebugGetStorage() map[string][]byte {
	return db.storage
}

func (db *memoryDatabase) GetReceiveBlockHeights(hash *types.Hash) ([]uint64, error) {
	return nil, nil
}

func (db *memoryDatabase) GetOneHourQuota() (uint64, error) {
	return 0, nil
}
