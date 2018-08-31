package vm

import (
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

type NoDatabase struct {
	balanceMap               map[types.Address]map[types.TokenTypeId]*big.Int
	storageMap               map[types.Address]map[types.Hash][]byte
	codeMap                  map[types.Address][]byte
	logList                  []*Log
	snapshotBlockMap         map[types.Hash]VmSnapshotBlock
	currentSnapshotBlockHash types.Hash
	accountBlockMap          map[types.Address]map[types.Hash]VmAccountBlock
	tokenMap                 map[types.TokenTypeId]VmToken
}

func NewNoDatabase() *NoDatabase {
	return &NoDatabase{
		balanceMap:       make(map[types.Address]map[types.TokenTypeId]*big.Int),
		storageMap:       make(map[types.Address]map[types.Hash][]byte),
		codeMap:          make(map[types.Address][]byte),
		logList:          make([]*Log, 0),
		snapshotBlockMap: make(map[types.Hash]VmSnapshotBlock),
		accountBlockMap:  make(map[types.Address]map[types.Hash]VmAccountBlock),
		tokenMap:         make(map[types.TokenTypeId]VmToken),
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
		db.balanceMap[addr][tokenId] = amount
	}

}
func (db *NoDatabase) SnapshotBlock(snapshotHash types.Hash) VmSnapshotBlock {
	if snapshotBlock, ok := db.snapshotBlockMap[snapshotHash]; ok {
		return snapshotBlock
	} else {
		return nil
	}

}
func (db *NoDatabase) SnapshotBlockByHeight(height *big.Int) VmSnapshotBlock {
	snapshotBlock := db.snapshotBlockMap[db.currentSnapshotBlockHash]
	for snapshotBlock != nil {
		if snapshotBlock.Height().Cmp(height) == 0 {
			return snapshotBlock
		} else if len(snapshotBlock.PrevHash().Bytes()) > 0 {
			snapshotBlock = db.snapshotBlockMap[snapshotBlock.PrevHash()]
		} else {
			return nil
		}
	}
	return nil
}
func (db *NoDatabase) SnapshotBlockList(startHeight *big.Int, count uint64, forward bool) []VmSnapshotBlock {
	// TODO
	return nil
}

func (db *NoDatabase) SnapshotHeight(snapshotHash types.Hash) *big.Int {
	if snapshotBlock, ok := db.snapshotBlockMap[snapshotHash]; ok {
		return new(big.Int).Set(snapshotBlock.Height())
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
	return ok
}
func (db *NoDatabase) IsExistToken(tokenId types.TokenTypeId) bool {
	// TODO
	return false
}
func (db *NoDatabase) CreateToken(tokenId types.TokenTypeId, tokenName string, owner types.Address, totelSupply *big.Int, decimals uint64) bool {
	if _, ok := db.tokenMap[tokenId]; !ok {
		db.tokenMap[tokenId] = VmToken{tokenId: tokenId, tokenName: tokenName, owner: owner, totalSupply: totelSupply, decimals: decimals}
		return true
	} else {
		return false
	}
}
func (db *NoDatabase) SetContractCode(addr types.Address, gid Gid, code []byte) {
	db.codeMap[addr] = code
	// TODO gid
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
	db.AddLog(log)
}
func (db *NoDatabase) LogListHash() types.Hash {
	return emptyHash
}

func (db *NoDatabase) IsExistGid(git Gid) bool {
	// TODO
	return true
}
