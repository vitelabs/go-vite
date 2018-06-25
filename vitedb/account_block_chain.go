package vitedb

import (
	"go-vite/ledger"
	"errors"
	"math/big"
	"bytes"
	"fmt"
)

type AccountBlockChain struct {
	db *DataBase
	accountStore *Account
}

func (bc AccountBlockChain) New () *AccountBlockChain {
	db := GetDataBase(DB_BLOCK)
	return &AccountBlockChain{
		db: db,
		accountStore: Account{}.New(),
	}
}

func (bc * AccountBlockChain) WriteBlock (block *ledger.AccountBlock) error {
	if (block.AccountAddress == nil) {
		return errors.New("Write block failed, because accountAddress is not exist")
	}


	accountMeta := bc.accountStore.GetAccountMeta(block.AccountAddress)

	lastAccountBlockHeight := big.NewInt(-2)

	if accountMeta != nil {
		for _, token := range accountMeta.TokenList {
			if bytes.Equal(token.TokenId, block.TokenId) {
				lastAccountBlockHeight = token.LastAccountBlockHeight
				break
			}
		}
	}

	fromHash := block.FromHash

	if fromHash == nil {
		if accountMeta == nil {
			return errors.New("Write block failed, because account is not exist")
		}

		// It is send block
		if lastAccountBlockHeight.Cmp(big.NewInt(-2)) == 0 {
			return errors.New("Write send block failed, because the account does not have this token")
		}

		lastAccountBlock, err := bc.GetBlockByHeight(lastAccountBlockHeight)

		if err != nil {
			return errors.New(fmt.Sprintln("Write send block failed, because GetBlockByHeight failed. Error is ", err))
		}

		if lastAccountBlock == nil || block.Amount.Cmp(lastAccountBlock.Balance) > 0 {
			return errors.New("Write send block failed, because the balance is not enough")
		}
	} else {
		// It is receive block
		fromBlockMeta, err:= bc.GetBlockMeta(block.FromHash)

		if fromBlockMeta == nil {
			return errors.New("Write receive block failed, because the from block is not exist")
		}

		if err != nil {
			return errors.New(fmt.Sprintln("Write receive block failed, because GetBlockByHeight failed. Error is ", err))
		}


		accountId := big.NewInt(123)

		if lastAccountBlockHeight.Cmp(big.NewInt(-2)) == 0 {
			// Write account meta
			if accountMeta == nil {
				accountMeta = &ledger.AccountMeta{
					AccountId: accountId,
				}
			}

			accountMeta.TokenList = append(accountMeta.TokenList, &ledger.AccountSimpleToken{
				TokenId: []byte{1, 2, 3},
				LastAccountBlockHeight: big.NewInt(-1),
			})

			bc.WriteAccountMeta(block.AccountAddress, accountMeta)
			bc.WriteAccountIdIndex(accountId, block.AccountAddress)
		}

		// Write self block meta
		bc.WriteAccountBlockMeta(block.Hash, &ledger.AccountBlockMeta{
			AccountId: accountId,
			Height: big.NewInt(456),
			Status: 2,
		})

		// Write from block meta
		fromBlockMeta.Status = 2 // Closed
		bc.WriteAccountBlockMeta(block.FromHash, fromBlockMeta)

	}

	// 模拟key, 需要改
	key :=  []byte("test")

	// Block serialize by protocol buffer
	data, err := block.DbSerialize()

	if err != nil {
		return errors.New(fmt.Sprintln("Write send block failed. Error is ", err))
	}

	bc.db.Put(key, data)

	return nil
}

func (bc * AccountBlockChain) WriteAccountMeta (accountAddress []byte, accountMeta *ledger.AccountMeta) error {
	return nil
}

func (bc * AccountBlockChain) WriteAccountIdIndex (accountId *big.Int, accountAddress []byte) error {
	return nil
}

func (bc * AccountBlockChain) WriteAccountBlockMeta (accountBlockHash []byte, accountBlockMeta *ledger.AccountBlockMeta) error {
	return nil
}


func (bc * AccountBlockChain) GetBlock (key []byte) (*ledger.AccountBlock, error) {
	//block, err := bc.db.Get(key)
	//if err != nil {
	//	fmt.Println(err)
	//	return nil, err
	//}
	//accountBlock := &ledger.AccountBlock{}
	//accountBlock.Deserialize(block)

	return nil, nil
}

func (bc * AccountBlockChain) GetBlockNumberByHash (account []byte, hash []byte) {

}

func (bc * AccountBlockChain) GetBlockByHeight (blockHeight *big.Int) (*ledger.AccountBlock, error) {
	return nil, nil
}


func (bc * AccountBlockChain) GetBlockMeta (blockHash []byte) (*ledger.AccountBlockMeta, error) {
	return nil, nil
}


func (bc * AccountBlockChain) Iterate (account []byte, startHash []byte, endHash []byte) {

}

func (bc * AccountBlockChain) RevertIterate (account []byte, startHash []byte, endHash []byte) {

}