package util

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"sort"
	"time"
)

var (
	AttovPerVite = big.NewInt(1e18)
)

func IsViteToken(tokenId types.TokenTypeId) bool {
	return tokenId == ledger.ViteTokenId
}
func IsSnapshotGid(gid types.Gid) bool {
	return gid == types.SNAPSHOT_GID
}

func MakeSendBlock(block *ledger.AccountBlock, toAddress types.Address, blockType byte, amount *big.Int, tokenId types.TokenTypeId, height uint64, data []byte) *ledger.AccountBlock {
	newTimestamp := time.Unix(0, block.Timestamp.UnixNano())
	return &ledger.AccountBlock{
		AccountAddress: block.AccountAddress,
		ToAddress:      toAddress,
		BlockType:      blockType,
		Amount:         amount,
		TokenId:        tokenId,
		Height:         height,
		SnapshotHash:   block.SnapshotHash,
		Data:           data,
		Fee:            big.NewInt(0),
		Timestamp:      &newTimestamp,
	}
}

func GetCreateContractData(bytecode []byte, gid types.Gid) []byte {
	return append(gid.Bytes(), bytecode...)
}

func GetGidFromCreateContractData(data []byte) types.Gid {
	gid, _ := types.BytesToGid(data[:types.GidSize])
	return gid
}

func GetCodeFromCreateContractData(data []byte) []byte {
	return data[types.GidSize:]
}

func NewContractAddress(accountAddress types.Address, accountBlockHeight uint64, prevBlockHash types.Hash, snapshotHash types.Hash) types.Address {
	return types.CreateContractAddress(
		accountAddress.Bytes(),
		new(big.Int).SetUint64(accountBlockHeight).Bytes(),
		prevBlockHash.Bytes(),
		snapshotHash.Bytes())
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
