package vm

import (
	"bytes"
	"encoding/hex"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/util"
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
	balanceMap        map[types.Address]map[types.TokenTypeId]*big.Int
	storageMap        map[types.Address]map[types.Hash][]byte
	codeMap           map[types.Address][]byte
	contractGidMap    map[types.Address]types.Gid
	logList           []Log
	snapshotBlockList []VmSnapshotBlock
	accountBlockMap   map[types.Address]map[types.Hash]VmAccountBlock
	tokenMap          map[types.TokenTypeId]VmToken
}

func NewNoDatabase() *NoDatabase {
	consensusGroupMap := make(map[types.Gid]consensusGroup)
	consensusGroupMap[util.SnapshotGid] = consensusGroup{}
	return &NoDatabase{
		balanceMap:        make(map[types.Address]map[types.TokenTypeId]*big.Int),
		storageMap:        make(map[types.Address]map[types.Hash][]byte),
		codeMap:           make(map[types.Address][]byte),
		contractGidMap:    make(map[types.Address]types.Gid),
		logList:           make([]Log, 0),
		snapshotBlockList: make([]VmSnapshotBlock, 0),
		accountBlockMap:   make(map[types.Address]map[types.Hash]VmAccountBlock),
		tokenMap:          make(map[types.TokenTypeId]VmToken),
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
		db.balanceMap[addr][tokenId] = new(big.Int).Sub(balance, amount)
	}
}
func (db *NoDatabase) AddBalance(addr types.Address, tokenId types.TokenTypeId, amount *big.Int) {
	if balance, ok := db.balanceMap[addr][tokenId]; ok {
		db.balanceMap[addr][tokenId] = new(big.Int).Add(balance, amount)
	} else {
		if _, ok := db.balanceMap[addr]; !ok {
			db.balanceMap[addr] = make(map[types.TokenTypeId]*big.Int)
		}
		db.balanceMap[addr][tokenId] = amount
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
func (db *NoDatabase) SetContractGid(addr types.Address, gid types.Gid, open bool) {}
func (db *NoDatabase) SetContractCode(addr types.Address, gid types.Gid, code []byte) {
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
			str += hex.EncodeToString(key.Bytes()) + "=>" + hex.EncodeToString(value) + ", "
		}
		return str
	} else {
		return ""
	}
}
func (db *NoDatabase) StorageHash(addr types.Address) types.Hash {
	return util.EmptyHash
}
func (db *NoDatabase) AddLog(log *Log) {
	db.logList = append(db.logList, *log)
}
func (db *NoDatabase) LogListHash() types.Hash {
	return util.EmptyHash
}

func (db *NoDatabase) GetDbIteratorByPrefix(prefix []byte) DbIterator {
	// TODO
	return nil
}

func prepareDb(viteTotalSupply *big.Int) (db *NoDatabase, addr1 types.Address, hash12 types.Hash, snapshot2 *NoSnapshotBlock, timestamp int64) {
	addr1, _ = types.BytesToAddress(util.HexToBytes("CA35B7D915458EF540ADE6068DFE2F44E8FA733C"))
	db = NewNoDatabase()
	db.tokenMap[util.ViteTokenTypeId] = VmToken{tokenId: util.ViteTokenTypeId, tokenName: "ViteToken", owner: addr1, totalSupply: viteTotalSupply, decimals: 18}

	timestamp = 1536214502
	snapshot1 := &NoSnapshotBlock{height: big.NewInt(1), timestamp: timestamp - 1, hash: types.DataHash([]byte{10, 1})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot1)
	snapshot2 = &NoSnapshotBlock{height: big.NewInt(2), timestamp: timestamp, hash: types.DataHash([]byte{10, 2})}
	db.snapshotBlockList = append(db.snapshotBlockList, snapshot2)

	hash11 := types.DataHash([]byte{1, 1})
	block11 := &NoAccountBlock{
		height:         big.NewInt(1),
		toAddress:      addr1,
		accountAddress: addr1,
		blockType:      BlockTypeSendCall,
		amount:         viteTotalSupply,
		tokenId:        util.ViteTokenTypeId,
		snapshotHash:   snapshot1.Hash(),
		depth:          1,
	}
	db.accountBlockMap[addr1] = make(map[types.Hash]VmAccountBlock)
	db.accountBlockMap[addr1][hash11] = block11
	hash12 = types.DataHash([]byte{1, 2})
	block12 := &NoAccountBlock{
		height:         big.NewInt(2),
		toAddress:      addr1,
		accountAddress: addr1,
		fromBlockHash:  hash11,
		blockType:      BlockTypeReceive,
		prevHash:       hash11,
		amount:         viteTotalSupply,
		tokenId:        util.ViteTokenTypeId,
		snapshotHash:   snapshot1.Hash(),
		depth:          1,
	}
	db.accountBlockMap[addr1][hash12] = block12

	db.balanceMap[addr1] = make(map[types.TokenTypeId]*big.Int)
	db.balanceMap[addr1][util.ViteTokenTypeId] = new(big.Int).Set(db.tokenMap[util.ViteTokenTypeId].totalSupply)

	db.storageMap[AddressConsensusGroup] = make(map[types.Hash][]byte)
	db.storageMap[AddressConsensusGroup][types.DataHash(util.SnapshotGid.Bytes())], _ = ABI_consensusGroup.PackVariable(VariableNameConsensusGroupInfo,
		uint8(25),
		int64(3),
		uint8(1),
		util.LeftPadBytes(util.ViteTokenTypeId.Bytes(), 32),
		uint8(1),
		util.JoinBytes(util.LeftPadBytes(registerAmount.Bytes(), 32), util.LeftPadBytes(util.ViteTokenTypeId.Bytes(), 32), util.LeftPadBytes(big.NewInt(registerLockTime).Bytes(), 32)),
		uint8(1),
		[]byte{})

	db.storageMap[AddressPledge] = make(map[types.Hash][]byte)
	db.storageMap[AddressPledge][types.DataHash(addr1.Bytes())], _ = ABI_pledge.PackVariable(VariableNamePledgeBeneficial, big.NewInt(1e18))
	return
}
