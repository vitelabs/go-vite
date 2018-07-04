package access

import (
	"github.com/vitelabs/go-vite/vitedb"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type AccountChainAccess struct {
	store *vitedb.AccountChain
	accountStore *vitedb.Account
	snapshotStore *vitedb.SnapshotChain
	tokenStore *vitedb.Token
}

var _accountChainAccess *AccountChainAccess

func GetAccountChainAccess () *AccountChainAccess {
	if _accountChainAccess == nil {
		_accountChainAccess = &AccountChainAccess {
			store: vitedb.GetAccountChain(),
			accountStore: vitedb.GetAccount(),
			snapshotStore: vitedb.GetSnapshotChain(),
			tokenStore: vitedb.GetToken(),
		}
	}

	return _accountChainAccess
}

func (aca *AccountChainAccess) GetBlockByHash (blockHash []byte) (*ledger.AccountBlock, error){
	accountBlock, err:= aca.store.GetBlockByHash(blockHash)
	if err != nil {
		return nil, err
	}

	if accountBlock.FromHash != nil{
		fromAccountBlockMeta, err := aca.store.GetBlockMeta(accountBlock.FromHash)

		if err != nil {
			return nil, err
		}


		fromAddress, err:= aca.accountStore.GetAddressById(fromAccountBlockMeta.AccountId)
		if err != nil {
			return nil, err
		}

		accountBlock.From = fromAddress
	}

	return accountBlock, nil
}

func (aca *AccountChainAccess) GetBlockListByAccountAddress (index int, num int, count int, accountAddress *types.Address) ([]*ledger.AccountBlock, error){
	accountMeta := aca.accountStore.GetAccountMeta(accountAddress)

	return aca.store.GetBlockListByAccountMeta(index, num, count, accountMeta)
}

func (aca *AccountChainAccess) GetBlockListByTokenId (index int, num int, count int, tokenId *types.TokenTypeId) ([]*ledger.AccountBlock, error){
	blockHashList, err := aca.tokenStore.GetAccountBlockHashListByTokenId(index, num, count, tokenId)
	if err != nil {
		return nil, err
	}
	var accountBlockList []*ledger.AccountBlock
	for _, blockHash := range blockHashList {
		block, err:= aca.store.GetBlockByHash(blockHash)
		if err != nil {
			return nil, err
		}
		accountBlockList = append(accountBlockList, block)
	}


	return accountBlockList, nil
}

func (aca *AccountChainAccess) GetBlockList (index, num, count int) ([]*ledger.AccountBlock, error) {
	blockHashList, err := aca.store.GetBlockHashList(index, num, count)
	if err != nil {
		return nil, err
	}

	var blockList []*ledger.AccountBlock
	for _,  blockHash := range blockHashList {
		block, err:= aca.GetBlockByHash(blockHash)
		if err != nil {
			return nil, err
		}
		blockList = append(blockList, block)
	}

	return blockList, nil
}

func (aca *AccountChainAccess) GetConfirmBlock (accountBlock *ledger.AccountBlock) (*ledger.SnapshotBlock, error) {
	var err error
	var confirmSnapshotBlock *ledger.SnapshotBlock

	aca.snapshotStore.Iterate(func(snapshotBlock *ledger.SnapshotBlock) bool {
		if itemAccountBlockHash, ok := snapshotBlock.Snapshot[accountBlock.AccountAddress.String()]; ok {
			var itemAccountBlockMeta *ledger.AccountBlockMeta
			itemAccountBlockMeta, err = aca.store.GetBlockMeta(itemAccountBlockHash)
			if itemAccountBlockMeta.Height.Cmp(accountBlock.Meta.Height) > 0 {
				confirmSnapshotBlock = snapshotBlock
				return false
			}
		}
		return false
	}, accountBlock.SnapshotTimestamp)

	return confirmSnapshotBlock, err

}

func (aca *AccountChainAccess) GetConfirmTimes (confirmSnapshotBlock *ledger.SnapshotBlock) (*big.Int, error) {
	if confirmSnapshotBlock == nil {
		return nil, nil
	}

	latestBlockHeight, err := aca.snapshotStore.GetLatestBlockHeight()
	if err != nil {
		return nil,  err
	}

	result := &big.Int{}
	result = result.Sub(latestBlockHeight, confirmSnapshotBlock.Height)
	return result, nil
}

func (aca *AccountChainAccess) GetAccountBalance (accountId *big.Int, blockHeight *big.Int) (*big.Int, error) {
	accountBLock, err := aca.store.GetBlockByHeight(accountId, blockHeight)
	if err != nil {
		return nil, err
	}
	return accountBLock.Balance, nil
}
