package access

import (
	"github.com/vitelabs/go-vite/vitedb"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/common/types"
)

type AccountChainAccess struct {
	store *vitedb.AccountChain
	accountStore *vitedb.Account
	tokenStore *vitedb.Token
}


var _accountChainAccess *AccountChainAccess

func GeAccountChainAccess () *AccountChainAccess {
	if _accountChainAccess == nil {
		_accountChainAccess = &AccountChainAccess {
			store: vitedb.GetAccountChain(),
			accountStore: vitedb.GetAccount(),
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

func (aca *AccountChainAccess) GetBlockList (index int, num int, count int) ([]*ledger.AccountBlock, error){
	return nil, nil
}