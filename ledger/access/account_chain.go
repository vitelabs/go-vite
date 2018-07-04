package access

import (
	"github.com/vitelabs/go-vite/vitedb"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"github.com/syndtr/goleveldb/leveldb"
	"errors"
	"bytes"
)

type AccountChainAccess struct {
	store         *vitedb.AccountChain
	accountStore  *vitedb.Account
	snapshotStore *vitedb.SnapshotChain
	tokenStore    *vitedb.Token
}

var _accountChainAccess *AccountChainAccess

func GetAccountChainAccess() *AccountChainAccess {
	if _accountChainAccess == nil {
		_accountChainAccess = &AccountChainAccess{
			store:         vitedb.GetAccountChain(),
			accountStore:  vitedb.GetAccount(),
			snapshotStore: vitedb.GetSnapshotChain(),
			tokenStore:    vitedb.GetToken(),
		}
	}

	return _accountChainAccess
}

func (aca *AccountChainAccess) WriteBlockList(blockList []*ledger.AccountBlock) error {
	batch := new(leveldb.Batch)
	var err error
	for _, block := range blockList {
		err = aca.writeBlock(batch, block)
		if err != nil {
			return err
		}
	}
	return nil
}

func (aca *AccountChainAccess) WriteBlock(block *ledger.AccountBlock) error {
	batch := new(leveldb.Batch)
	return aca.store.BatchWrite(batch, func(batch *leveldb.Batch) error {
		return aca.writeBlock(batch, block)
	})
}

func (aca *AccountChainAccess) writeBlock(batch *leveldb.Batch, block *ledger.AccountBlock) error {

	accountMeta := aca.accountStore.GetAccountMeta(block.AccountAddress)

	var lastAccountToken *ledger.AccountSimpleToken
	if accountMeta != nil {
		for _, token := range accountMeta.TokenList {
			if token.TokenId.String() == block.TokenId.String() {
				lastAccountToken = token
				break
			}
		}
	}

	if block.FromHash == nil {
		// send block
		if accountMeta == nil {
			return errors.New("Write send block failed, because the account does not exist.")
		}


		if lastAccountToken == nil {
			return errors.New("Write send block failed, because the account does not have this token")
		}

		lastAccountBlock, err := aca.store.GetBlockByHeight(accountMeta.AccountId, lastAccountToken.LastAccountBlockHeight)
		if err != nil {
			return err
		}

		if lastAccountBlock == nil || block.Amount.Cmp(lastAccountBlock.Balance) > 0 {
			return errors.New("Write send block failed, because the balance is not enough")
		}
	} else {
		// receive block
		if accountMeta == nil {
			accountMeta = &ledger.AccountMeta{}
			// Write account meta
			// Write account index
		}

		if bytes.Equal(block.To.Bytes(), []byte{0}) {
			mintage, err := ledger.NewMintage(block)
			if err != nil {
				return err
			}

			// Write Mintage
			if err := aca.tokenStore.WriteTokenIdIndex(batch, mintage.Id, big.NewInt(0), block.Hash); err != nil{
				return err
			}

			// Write TokenName body
			if err := aca.tokenStore.WriteTokenNameIndex(batch, mintage.Name, mintage.Id); err != nil{
				return err
			}

			// Write TokenSymbol body
			if err := aca.tokenStore.WriteTokenSymbolIndex(batch, mintage.Symbol, mintage.Id); err != nil{
				return err
			}
		}
	}

	// aca.accountStore.WriteAccountMeta()
	latestBlockHeight, err := aca.store.GetLatestBlockHeightByAccountId(accountMeta.AccountId)
	if err != nil {
		return err
	}

	newBlockHeight := latestBlockHeight.Add(latestBlockHeight, big.NewInt(1))

	if err := aca.store.WriteBlock(batch, accountMeta.AccountId, newBlockHeight, block); err != nil {
		return err
	}


	// write block meta
	newBlockMeta := &ledger.AccountBlockMeta {
		Height: newBlockHeight,
		AccountId: accountMeta.AccountId,
	}

	if block.FromHash == nil {
		newBlockMeta.Status = 1 // open
	} else {
		newBlockMeta.Status = 2 // closed
		fromBlockMeta, err := aca.store.GetBlockMeta(block.FromHash)
		if fromBlockMeta == nil {
			return errors.New("Write receive block failed, because the from block is not exist")
		}

		if err != nil {
			return err
		}

		fromBlockMeta.Status = 2 // closed

		if err := aca.store.WriteBlockMeta(batch, block.FromHash, fromBlockMeta); err != nil {
			return err
		}
	}


	if err := aca.store.WriteBlockMeta(batch, block.Hash, newBlockMeta); err != nil {
		return err
	}


	// Write TokenId index
	latestBlockHeightInToken, err := aca.tokenStore.GetLatestBlockHeightByTokenId(block.TokenId)
	if err != nil {
		return err
	}
	newBlockHeightInToken := latestBlockHeightInToken.Add(latestBlockHeightInToken, big.NewInt(1))

	if err := aca.tokenStore.WriteTokenIdIndex(batch, block.TokenId, newBlockHeightInToken, block.Hash); err != nil {
		return err
	}

	// Write st index
	stHeight, err := aca.snapshotStore.GetHeightByHash(block.SnapshotTimestamp)
	if err != nil {
		return err
	}

	lastStId, err := aca.store.GetLastIdByStHeight(stHeight)
	if err != nil {
		return err
	}

	lastStId.Add(lastStId, big.NewInt(1))
	if err := aca.store.WriteStIndex(batch, block.SnapshotTimestamp, lastStId, block.Hash); err != nil {
		return err
	}
	return nil
}

func (aca *AccountChainAccess) GetBlockByHash(blockHash []byte) (*ledger.AccountBlock, error) {
	accountBlock, err := aca.store.GetBlockByHash(blockHash)
	if err != nil {
		return nil, err
	}

	if accountBlock.FromHash != nil {
		fromAccountBlockMeta, err := aca.store.GetBlockMeta(accountBlock.FromHash)

		if err != nil {
			return nil, err
		}

		fromAddress, err := aca.accountStore.GetAddressById(fromAccountBlockMeta.AccountId)
		if err != nil {
			return nil, err
		}

		accountBlock.From = fromAddress
	}

	return accountBlock, nil
}

func (aca *AccountChainAccess) GetBlockListByAccountAddress(index int, num int, count int, accountAddress *types.Address) ([]*ledger.AccountBlock, error) {
	accountMeta := aca.accountStore.GetAccountMeta(accountAddress)

	return aca.store.GetBlockListByAccountMeta(index, num, count, accountMeta)
}

func (aca *AccountChainAccess) GetBlockListByTokenId(index int, num int, count int, tokenId *types.TokenTypeId) ([]*ledger.AccountBlock, error) {
	blockHashList, err := aca.tokenStore.GetAccountBlockHashListByTokenId(index, num, count, tokenId)
	if err != nil {
		return nil, err
	}
	var accountBlockList []*ledger.AccountBlock
	for _, blockHash := range blockHashList {
		block, err := aca.store.GetBlockByHash(blockHash)
		if err != nil {
			return nil, err
		}
		accountBlockList = append(accountBlockList, block)
	}

	return accountBlockList, nil
}

func (aca *AccountChainAccess) GetBlockList(index, num, count int) ([]*ledger.AccountBlock, error) {
	blockHashList, err := aca.store.GetBlockHashList(index, num, count)
	if err != nil {
		return nil, err
	}

	var blockList []*ledger.AccountBlock
	for _, blockHash := range blockHashList {
		block, err := aca.GetBlockByHash(blockHash)
		if err != nil {
			return nil, err
		}
		blockList = append(blockList, block)
	}

	return blockList, nil
}

func (aca *AccountChainAccess) GetConfirmBlock(accountBlock *ledger.AccountBlock) (*ledger.SnapshotBlock, error) {
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

func (aca *AccountChainAccess) GetConfirmTimes(confirmSnapshotBlock *ledger.SnapshotBlock) (*big.Int, error) {
	if confirmSnapshotBlock == nil {
		return nil, nil
	}

	latestBlockHeight, err := aca.snapshotStore.GetLatestBlockHeight()
	if err != nil {
		return nil, err
	}

	result := &big.Int{}
	result = result.Sub(latestBlockHeight, confirmSnapshotBlock.Height)
	return result, nil
}
