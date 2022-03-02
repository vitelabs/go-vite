package chain_genesis

import (
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
)

const (
	LedgerUnknown = byte(0)
	LedgerEmpty   = byte(1)
	LedgerValid   = byte(2)
	LedgerInvalid = byte(3)
)

func InitLedger(chain Chain, genesisSnapshotBlock *ledger.SnapshotBlock, vmBlocks []*interfaces.VmAccountBlock) error {
	// insert genesis account blocks
	for _, ab := range vmBlocks {
		err := chain.InsertAccountBlock(ab)
		if err != nil {
			panic(err)
		}
	}

	// insert genesis snapshot block
	chain.InsertSnapshotBlock(genesisSnapshotBlock)

	sumHash := CheckSum(vmBlocks)
	return chain.WriteGenesisCheckSum(sumHash)
}

func CheckLedger(chain Chain, genesisSnapshotBlock *ledger.SnapshotBlock, genesisAccountBlocks []*interfaces.VmAccountBlock) (byte, error) {
	firstSb, err := chain.QuerySnapshotBlockByHeight(1)
	if err != nil {
		return LedgerUnknown, err
	}
	if firstSb == nil {
		return LedgerEmpty, nil
	}

	if firstSb.Hash != genesisSnapshotBlock.Hash {
		return LedgerInvalid, nil
	}

	sumHash := CheckSum(genesisAccountBlocks)

	querySumHash, err := chain.QueryGenesisCheckSum()
	if err != nil {
		return LedgerUnknown, err
	}
	if querySumHash == nil {
		if err := chain.WriteGenesisCheckSum(sumHash); err != nil {
			return LedgerUnknown, err

		}
	} else if sumHash != *querySumHash {
		return LedgerInvalid, nil
	}

	return LedgerValid, nil
}

func VmBlocksToHashMap(accountBlocks []*interfaces.VmAccountBlock) map[types.Hash]struct{} {
	hashMap := make(map[types.Hash]struct{}, len(accountBlocks))

	for _, accountBlock := range accountBlocks {
		hashMap[accountBlock.AccountBlock.Hash] = struct{}{}
	}
	return hashMap
}
