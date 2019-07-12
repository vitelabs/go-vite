package util

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/abi"
	"math/big"
	"sort"
)

var (
	AttovPerVite                    = big.NewInt(1e18)
	CreateContractDataLengthMin     = 13
	CreateContractDataLengthMinRand = 14
)

func IsViteToken(tokenId types.TokenTypeId) bool {
	return tokenId == ledger.ViteTokenId
}
func IsSnapshotGid(gid types.Gid) bool {
	return gid == types.SNAPSHOT_GID
}
func IsDelegateGid(gid types.Gid) bool {
	return gid == types.DELEGATE_GID
}

func MakeSendBlock(fromAddress types.Address, toAddress types.Address, blockType byte, amount *big.Int, tokenId types.TokenTypeId, data []byte) *ledger.AccountBlock {
	return &ledger.AccountBlock{
		AccountAddress: fromAddress,
		ToAddress:      toAddress,
		BlockType:      blockType,
		Amount:         amount,
		TokenId:        tokenId,
		Data:           data,
		Fee:            big.NewInt(0),
	}
}

var (
	SolidityPPContractType uint8 = 1
	contractTypeSize             = 1
	confirmTimeSize              = 1
	seedCountSize                = 1
	quotaRatioSize               = 1
)

func IsValidQuotaRatio(quotaRatio uint8) bool {
	return quotaRatio >= 10 && quotaRatio <= 100
}

func GetCreateContractData(bytecode []byte, contractType uint8, confirmTimes uint8, seedCount uint8, quotaRatio uint8, gid types.Gid, snapshotHeight uint64) []byte {
	if !fork.IsSeedFork(snapshotHeight) {
		return helper.JoinBytes(gid.Bytes(), []byte{contractType}, []byte{confirmTimes}, []byte{quotaRatio}, bytecode)
	} else {
		return helper.JoinBytes(gid.Bytes(), []byte{contractType}, []byte{confirmTimes}, []byte{seedCount}, []byte{quotaRatio}, bytecode)
	}
}

func GetGidFromCreateContractData(data []byte) types.Gid {
	gid, _ := types.BytesToGid(data[:types.GidSize])
	return gid
}

func GetContractTypeFromCreateContractData(data []byte) uint8 {
	return uint8(data[types.GidSize])
}
func IsExistContractType(contractType uint8) bool {
	if contractType == SolidityPPContractType {
		return true
	}
	return false
}
func GetConfirmTimeFromCreateContractData(data []byte) uint8 {
	return uint8(data[types.GidSize+contractTypeSize])
}
func GetSeedCountFromCreateContractData(data []byte) uint8 {
	return uint8(data[types.GidSize+contractTypeSize+confirmTimeSize])
}
func GetQuotaRatioFromCreateContractData(data []byte, snapshotHeight uint64) uint8 {
	if !fork.IsSeedFork(snapshotHeight) {
		return uint8(data[types.GidSize+contractTypeSize+confirmTimeSize])
	}
	return uint8(data[types.GidSize+contractTypeSize+confirmTimeSize+seedCountSize])
}
func GetCodeFromCreateContractData(data []byte, snapshotHeight uint64) []byte {
	if !fork.IsSeedFork(snapshotHeight) {
		return data[types.GidSize+contractTypeSize+confirmTimeSize+quotaRatioSize:]
	}
	return data[types.GidSize+contractTypeSize+confirmTimeSize+seedCountSize+quotaRatioSize:]
}
func PackContractCode(contractType uint8, code []byte) []byte {
	return helper.JoinBytes([]byte{contractType}, code)
}

type CommonDb interface {
	Address() *types.Address
	IsContractAccount() (bool, error)
	GetContractCode() ([]byte, error)
	GetContractCodeBySnapshotBlock(addr *types.Address, snapshotBlock *ledger.SnapshotBlock) ([]byte, error)
}

func GetContractCode(db CommonDb, addr *types.Address, status GlobalStatus) ([]byte, []byte) {
	var code []byte
	var err error
	if *db.Address() == *addr {
		code, err = db.GetContractCode()
	} else {
		code, err = db.GetContractCodeBySnapshotBlock(addr, status.SnapshotBlock())
	}
	DealWithErr(err)
	if len(code) > 0 {
		return code[:contractTypeSize], code[contractTypeSize:]
	}
	return nil, nil
}

func NewContractAddress(accountAddress types.Address, accountBlockHeight uint64, prevBlockHash types.Hash) types.Address {
	return types.CreateContractAddress(
		accountAddress.Bytes(),
		helper.LeftPadBytes(new(big.Int).SetUint64(accountBlockHeight).Bytes(), 8),
		prevBlockHash.Bytes())
}

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

func IsUserAccount(addr types.Address) bool {
	return !types.IsContractAddr(addr)
}

func NewLog(c abi.ABIContract, name string, params ...interface{}) *ledger.VmLog {
	topics, data, _ := c.PackEvent(name, params...)
	return &ledger.VmLog{Topics: topics, Data: data}
}
