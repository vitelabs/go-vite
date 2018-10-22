package util

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
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
