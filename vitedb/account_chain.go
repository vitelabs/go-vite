package vitedb

import (
	"errors"
	"math/big"
	"bytes"
	"fmt"
	"github.com/vitelabs/go-vite/ledger"
)

type AccountChain struct {
	db *DataBase
	accountStore *Account
}

var _accountchain *AccountChain
func (ac AccountChain) GetInstance () *AccountChain {
	db := GetDataBase(DB_BLOCK)
	if _accountchain == nil {
		_accountchain = &AccountChain{
			db: db,
			accountStore: Account{}.New(),
		}
	}


	return _accountchain
}

func (ac * AccountChain) WriteBlock (block *ledger.AccountBlock) error {
	if (block.AccountAddress == nil) {
		return errors.New("Write block failed, because accountAddress is not exist")
	}


	accountMeta := ac.accountStore.GetAccountMeta(block.AccountAddress)

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

		lastAccountBlock, err := ac.GetBlockByHeight(lastAccountBlockHeight)

		if err != nil {
			return errors.New(fmt.Sprintln("Write send block failed, because GetBlockByHeight failed. Error is ", err))
		}

		if lastAccountBlock == nil || block.Amount.Cmp(lastAccountBlock.Balance) > 0 {
			return errors.New("Write send block failed, because the balance is not enough")
		}
	} else {
		// It is receive block
		fromBlockMeta, err:= ac.GetBlockMeta(block.FromHash)

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

			ac.WriteAccountMeta(block.AccountAddress, accountMeta)
			ac.WriteAccountIdIndex(accountId, block.AccountAddress)
		}

		// Write self block meta
		ac.WriteAccountBlockMeta(block.Hash, &ledger.AccountBlockMeta{
			AccountId: accountId,
			Height: big.NewInt(456),
			Status: 2,
		})

		// Write from block meta
		fromBlockMeta.Status = 2 // Closed
		ac.WriteAccountBlockMeta(block.FromHash, fromBlockMeta)

	}

	// 模拟key, 需要改
	key :=  []byte("test")

	// Block serialize by protocol buffer
	data, err := block.DbSerialize()

	if err != nil {
		return errors.New(fmt.Sprintln("Write send block failed. Error is ", err))
	}

	ac.db.Put(key, data)

	return nil
}

func (ac * AccountChain) WriteSendBlock () error {


	return nil
}

func (ac * AccountChain) WriteReceiveBlock () error {


	return nil
}

func (ac * AccountChain) WriteAccountMeta (accountAddress []byte, accountMeta *ledger.AccountMeta) error {

	return nil
}

func (ac * AccountChain) WriteAccountIdIndex (accountId *big.Int, accountAddress []byte) error {

	return nil
}

func (ac * AccountChain) WriteAccountBlockMeta (accountBlockHash []byte, accountBlockMeta *ledger.AccountBlockMeta) error {

	return nil
}


func (ac * AccountChain) GetBlockByBlockHash (blockHash []byte) (*ledger.AccountBlock, error) {
	//block, err := ac.db.Get(key)
	//if err != nil {
	//	fmt.Println(err)
	//	return nil, err
	//}
	//accountBlock := &ledger.AccountBlock{}
	//accountBlock.Deserialize(block)

	return nil, nil
}



func (ac * AccountChain) GetBlockNumberByHash (account []byte, hash []byte) {

}

func (ac * AccountChain) GetBlockByHeight (blockHeight *big.Int) (*ledger.AccountBlock, error) {
	return nil, nil
}


func (ac * AccountChain) GetBlockMeta (blockHash []byte) (*ledger.AccountBlockMeta, error) {
	return nil, nil
}


func (ac * AccountChain) Iterate (account []byte, startHash []byte, endHash []byte) {

}

func (ac * AccountChain) RevertIterate (account []byte, startHash []byte, endHash []byte) {

}