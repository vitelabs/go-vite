package vm

import (
	"bytes"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type VmToken struct {
	tokenId     types.TokenTypeId
	tokenName   string
	owner       types.Address
	totalSupply *big.Int
	decimals    uint64
}

type consensusGroup struct{}

type NoDatabase struct {
	balanceMap        map[types.Address]map[types.TokenTypeId]*big.Int
	storageMap        map[types.Address]map[types.Hash][]byte
	codeMap           map[types.Address][]byte
	contractGidMap    map[types.Address]Gid
	logList           []Log
	snapshotBlockList []VmSnapshotBlock
	accountBlockMap   map[types.Address]map[types.Hash]VmAccountBlock
	tokenMap          map[types.TokenTypeId]VmToken
	consensusGroupMap map[Gid]consensusGroup
}

func NewNoDatabase() *NoDatabase {
	consensusGroupMap := make(map[Gid]consensusGroup)
	consensusGroupMap[snapshotGid] = consensusGroup{}
	return &NoDatabase{
		balanceMap:        make(map[types.Address]map[types.TokenTypeId]*big.Int),
		storageMap:        make(map[types.Address]map[types.Hash][]byte),
		codeMap:           make(map[types.Address][]byte),
		contractGidMap:    make(map[types.Address]Gid),
		logList:           make([]Log, 0),
		snapshotBlockList: make([]VmSnapshotBlock, 0),
		accountBlockMap:   make(map[types.Address]map[types.Hash]VmAccountBlock),
		tokenMap:          make(map[types.TokenTypeId]VmToken),
		consensusGroupMap: consensusGroupMap,
	}
}

func (db *NoDatabase) Balance(addr types.Address, tokenId types.TokenTypeId) *big.Int {
	if balance, ok := db.balanceMap[addr][tokenId]; ok {
		return new(big.Int).Set(balance)
	} else {
		return big.NewInt(0)
	}
}
func (db *NoDatabase) SubBalance(addr types.Address, tokenId types.TokenTypeId, amount *big.Int) {
	balance, ok := db.balanceMap[addr][tokenId]
	if ok && balance.Cmp(amount) >= 0 {
		balance.Sub(balance, amount)
	}
}
func (db *NoDatabase) AddBalance(addr types.Address, tokenId types.TokenTypeId, amount *big.Int) {
	if balance, ok := db.balanceMap[addr][tokenId]; ok {
		balance.Add(balance, amount)
	} else {
		if _, ok := db.balanceMap[addr]; !ok {
			db.balanceMap[addr] = make(map[types.TokenTypeId]*big.Int)
		}
		db.balanceMap[addr][tokenId] = new(big.Int).Set(amount)
	}

}
func (db *NoDatabase) SnapshotBlock(snapshotHash types.Hash) VmSnapshotBlock {
	for len := len(db.snapshotBlockList) - 1; len >= 0; len = len - 1 {
		block := db.snapshotBlockList[len]
		if bytes.Equal(block.Hash().Bytes(), snapshotHash.Bytes()) {
			return block
		}
	}
	return nil

}
func (db *NoDatabase) SnapshotBlockByHeight(height *big.Int) VmSnapshotBlock {
	if int(height.Int64()) < len(db.snapshotBlockList) {
		return db.snapshotBlockList[height.Int64()-1]
	}
	return nil
}
func (db *NoDatabase) SnapshotBlockList(startHeight *big.Int, count uint64, forward bool) []VmSnapshotBlock {
	if forward {
		start := startHeight.Uint64()
		end := start + count
		return db.snapshotBlockList[start:end]
	} else {
		end := startHeight.Uint64() - 1
		start := end - count
		return db.snapshotBlockList[start:end]
	}
}

func (db *NoDatabase) SnapshotHeight(snapshotHash types.Hash) *big.Int {
	block := db.SnapshotBlock(snapshotHash)
	if block != nil {
		return block.Height()
	} else {
		return big.NewInt(0)
	}
}
func (db *NoDatabase) AccountBlock(addr types.Address, blockHash types.Hash) VmAccountBlock {
	if VmAccountBlock, ok := db.accountBlockMap[addr][blockHash]; ok {
		return VmAccountBlock
	} else {
		return nil
	}
}
func (db *NoDatabase) Rollback() {}
func (db *NoDatabase) IsExistAddress(addr types.Address) bool {
	_, ok := db.accountBlockMap[addr]
	if !ok {
		_, ok := getPrecompiledContract(addr)
		return ok
	}
	return ok
}
func (db *NoDatabase) IsExistToken(tokenId types.TokenTypeId) bool {
	_, ok := db.tokenMap[tokenId]
	return ok
}
func (db *NoDatabase) CreateToken(tokenId types.TokenTypeId, tokenName string, owner types.Address, totelSupply *big.Int, decimals uint64) bool {
	if _, ok := db.tokenMap[tokenId]; !ok {
		db.tokenMap[tokenId] = VmToken{tokenId: tokenId, tokenName: tokenName, owner: owner, totalSupply: totelSupply, decimals: decimals}
		return true
	} else {
		return false
	}
}
func (db *NoDatabase) SetContractGid(addr types.Address, gid Gid, open bool) {}
func (db *NoDatabase) SetContractCode(addr types.Address, gid Gid, code []byte) {
	db.codeMap[addr] = code
	db.contractGidMap[addr] = gid
}
func (db *NoDatabase) ContractCode(addr types.Address) []byte {
	if code, ok := db.codeMap[addr]; ok {
		return code
	} else {
		return nil
	}
}
func (db *NoDatabase) Storage(addr types.Address, loc types.Hash) []byte {
	if data, ok := db.storageMap[addr][loc]; ok {
		return data
	} else {
		return []byte{}
	}
}
func (db *NoDatabase) SetStorage(addr types.Address, loc types.Hash, value []byte) {
	if _, ok := db.storageMap[addr]; !ok {
		db.storageMap[addr] = make(map[types.Hash][]byte)
	}
	db.storageMap[addr][loc] = value
}
func (db *NoDatabase) PrintStorage(addr types.Address) string {
	if storage, ok := db.storageMap[addr]; ok {
		var str string
		for key, value := range storage {
			str += hexToString(key.Bytes()) + " => " + hexToString(value) + "\n"
		}
		return str
	} else {
		return ""
	}
}
func (db *NoDatabase) StorageHash(addr types.Address) types.Hash {
	return emptyHash
}
func (db *NoDatabase) AddLog(log *Log) {
	db.logList = append(db.logList, *log)
}
func (db *NoDatabase) LogListHash() types.Hash {
	return emptyHash
}

func (db *NoDatabase) IsExistGid(gid Gid) bool {
	_, ok := db.consensusGroupMap[gid]
	return ok
}
