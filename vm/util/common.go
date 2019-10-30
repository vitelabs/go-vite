package util

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"sort"
	"unicode"
)

var (
	// AttovPerVite defines decimals of vite
	AttovPerVite = big.NewInt(1e18)
	// CreateContractDataLengthMin defines create contract request block data prefix length before seed fork
	CreateContractDataLengthMin = 13
	// CreateContractDataLengthMinRand defines create contract request block data prefix length after seed fork
	CreateContractDataLengthMinRand = 14
)

// IsViteToken checks whether tokenId is vite token
func IsViteToken(tokenID types.TokenTypeId) bool {
	return tokenID == ledger.ViteTokenId
}

// IsSnapshotGid checks whether gid is snapshot consensus group id
func IsSnapshotGid(gid types.Gid) bool {
	return gid == types.SNAPSHOT_GID
}

// IsDelegateGid checks whether gid is global delegate consensus group id
func IsDelegateGid(gid types.Gid) bool {
	return gid == types.DELEGATE_GID
}

// MakeRequestBlock returns a request block
func MakeRequestBlock(fromAddress types.Address, toAddress types.Address, blockType byte, amount *big.Int, tokenID types.TokenTypeId, data []byte) *ledger.AccountBlock {
	return &ledger.AccountBlock{
		AccountAddress: fromAddress,
		ToAddress:      toAddress,
		BlockType:      blockType,
		Amount:         amount,
		TokenId:        tokenID,
		Data:           data,
		Fee:            big.NewInt(0),
	}
}

var (
	// SolidityPPContractType defines contract type of solidity++ byte code
	SolidityPPContractType    uint8 = 1
	contractTypeSize                = 1
	snapshotCountSize               = 1
	snapshotWithSeedCountSize       = 1
	quotaMultiplierSize             = 1
)

// GetCreateContractData generate create contract request block data
func GetCreateContractData(bytecode []byte, contractType uint8, snapshotCount uint8, snapshotWithSeedCount uint8, quotaMultiplier uint8, gid types.Gid) []byte {
	return helper.JoinBytes(gid.Bytes(), []byte{contractType}, []byte{snapshotCount}, []byte{snapshotWithSeedCount}, []byte{quotaMultiplier}, bytecode)
}

// GetGidFromCreateContractData decode gid from create contract request block data
func GetGidFromCreateContractData(data []byte) types.Gid {
	gid, _ := types.BytesToGid(data[:types.GidSize])
	return gid
}

// GetContractTypeFromCreateContractData decode contract type from create contract request block data
func GetContractTypeFromCreateContractData(data []byte) uint8 {
	return uint8(data[types.GidSize])
}

// IsExistContractType check contract type validation
func IsExistContractType(contractType uint8) bool {
	if contractType == SolidityPPContractType {
		return true
	}
	return false
}

// GetSnapshotCountFromCreateContractData decode snapshot block count(response latency) from create contract request block data
func GetSnapshotCountFromCreateContractData(data []byte) uint8 {
	return uint8(data[types.GidSize+contractTypeSize])
}

// GetSnapshotWithSeedCountCountFromCreateContractData decode snapshot block with seed count(random degree) from create contract request block data
func GetSnapshotWithSeedCountCountFromCreateContractData(data []byte) uint8 {
	return uint8(data[types.GidSize+contractTypeSize+snapshotCountSize])
}

// GetQuotaMultiplierFromCreateContractData decode quota multiplier from create contract request block data
func GetQuotaMultiplierFromCreateContractData(data []byte, snapshotHeight uint64) uint8 {
	if !fork.IsSeedFork(snapshotHeight) {
		return uint8(data[types.GidSize+contractTypeSize+snapshotCountSize])
	}
	return uint8(data[types.GidSize+contractTypeSize+snapshotCountSize+snapshotWithSeedCountSize])
}

// GetCodeFromCreateContractData decode code and constructor params from create contract request block data
func GetCodeFromCreateContractData(data []byte, snapshotHeight uint64) []byte {
	if !fork.IsSeedFork(snapshotHeight) {
		return data[types.GidSize+contractTypeSize+snapshotCountSize+quotaMultiplierSize:]
	}
	return data[types.GidSize+contractTypeSize+snapshotCountSize+snapshotWithSeedCountSize+quotaMultiplierSize:]
}

// PackContractCode generate contract code on chain
func PackContractCode(contractType uint8, code []byte) []byte {
	return helper.JoinBytes([]byte{contractType}, code)
}

// NewContractAddress generate contract address in create contract request block
func NewContractAddress(accountAddress types.Address, accountBlockHeight uint64, prevBlockHash types.Hash) types.Address {
	return types.CreateContractAddress(
		accountAddress.Bytes(),
		helper.LeftPadBytes(new(big.Int).SetUint64(accountBlockHeight).Bytes(), 8),
		prevBlockHash.Bytes())
}

// PrintMap is used to print storage map under debug mode
func PrintMap(m map[string][]byte) string {
	var result string
	if len(m) > 0 {
		var keys []string
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			result += hex.EncodeToString([]byte(k)) + "=>" + hex.EncodeToString(m[k]) + ", "
		}
		result = result[:len(result)-2]
	}
	return result
}

// CheckFork check whether current snapshot block height is over certain hard fork
func CheckFork(db dbInterface, f func(uint64) bool) bool {
	sb, err := db.LatestSnapshotBlock()
	DealWithErr(err)
	return f(sb.Height)
}

// FirstToLower change first character for string to lower case
func FirstToLower(str string) string {
	return string(unicode.ToLower(rune(str[0]))) + str[1:]
}

func ComputeSendBlockHash(receiveBlock *ledger.AccountBlock, sendBlock *ledger.AccountBlock, index uint8) types.Hash {
	return sendBlock.ComputeSendHash(receiveBlock, index)
}
