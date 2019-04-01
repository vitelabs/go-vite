package chain_state

import (
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

// TODO
func (sDB *StateDB) Rollback(deletedSnapshotSegments []*chain_block.SnapshotSegment) error {
	batch := sDB.store.NewBatch()
	//blockHashList := make([]*types.Hash, 0, size)

	allBalanceMap := make(map[types.Address]map[types.TokenTypeId]*big.Int)
	getBalance := func(addr types.Address, tokenTypeId types.TokenTypeId) (*big.Int, error) {
		balanceMap, ok := allBalanceMap[addr]
		if !ok {
			balanceMap = make(map[types.TokenTypeId]*big.Int)
			allBalanceMap[addr] = balanceMap
		}

		balance, ok := balanceMap[tokenTypeId]
		if !ok {
			var err error
			balance, err = sDB.chain.GetBalance(addr, tokenTypeId)
			if err != nil {
				return nil, err
			}
			balanceMap[tokenTypeId] = balance

		}
		return balance, nil

	}
	isDeleteSnapshotBlock := false
	for _, seg := range deletedSnapshotSegments {
		snapshotBlock := seg.SnapshotBlock
		if snapshotBlock != nil {
			isDeleteSnapshotBlock = true
		}

		deleteKey := make(map[string]struct{})

		for _, accountBlock := range seg.AccountBlocks {
			// todo rollback key value

			// rollback balance
			addr := accountBlock.AccountAddress
			tokenId := accountBlock.TokenId

			var sendBlock *ledger.AccountBlock

			if accountBlock.IsReceiveBlock() {
				sendBlock, err := sDB.chain.GetAccountBlockByHash(accountBlock.FromBlockHash)
				if err != nil {
					return err
				}
				tokenId = sendBlock.TokenId
			}
			balance, err := getBalance(addr, tokenId)
			if err != nil {
				return err
			}
			if accountBlock.IsReceiveBlock() {
				balance.Add(balance, sendBlock.Amount)
			} else {
				balance.Sub(balance, accountBlock.Amount)

			}
			allBalanceMap[addr][tokenId] = balance

			// delete history balance
			if snapshotBlock != nil {
				deleteKey[string(chain_utils.CreateHistoryBalanceKey(addr, tokenId, snapshotBlock.Height))] = struct{}{}
			}

			// delete code
			if accountBlock.Height <= 1 {
				batch.Delete(chain_utils.CreateCodeKey(accountBlock.AccountAddress))
			}

			// delete contract meta
			if accountBlock.BlockType == ledger.BlockTypeSendCreate {
				batch.Delete(chain_utils.CreateContractMetaKey(accountBlock.AccountAddress))
			}

			// delete log hash
			if accountBlock.LogHash != nil {
				batch.Delete(chain_utils.CreateVmLogListKey(accountBlock.LogHash))
			}

			// delete call depth
			if accountBlock.IsReceiveBlock() {
				for _, sendBlock := range accountBlock.SendBlockList {
					batch.Delete(chain_utils.CreateCallDepthKey(&sendBlock.Hash))
				}
			}
		}

		for key := range deleteKey {
			batch.Delete([]byte(key))
		}
	}

	// reset index
	for addr, balanceMap := range allBalanceMap {
		for tokenTypeId, balance := range balanceMap {
			balanceBytes := balance.Bytes()
			batch.Put(chain_utils.CreateBalanceKey(addr, tokenTypeId), balanceBytes)
			if !isDeleteSnapshotBlock {
				batch.Put(chain_utils.CreateHistoryBalanceKey(addr, tokenTypeId, sDB.chain.GetLatestSnapshotBlock().Height+1), balanceBytes)
			}
		}
	}

	sDB.store.Write(batch)
	return nil
}
