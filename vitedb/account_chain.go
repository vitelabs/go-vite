package vitedb

import (
	"math/big"
	"github.com/vitelabs/go-vite/ledger"
	"log"
	"github.com/syndtr/goleveldb/leveldb"
	"bytes"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type AccountChain struct {
	db *DataBase
}

var _accountchain *AccountChain
func GetAccountChain () *AccountChain {
	db, err := GetLDBDataBase(DB_BLOCK)
	if err != nil {
		log.Fatal(err)
	}

	if _accountchain == nil {
		_accountchain = &AccountChain{
			db: db,
		}
	}


	return _accountchain
}

func (ac * AccountChain) WriteBlock (batch *leveldb.Batch, block *ledger.AccountBlock) error {
	return batchWrite(batch, ac.db.Leveldb, func(context *batchContext) error {
		if block.FromHash == nil {
			// Send block
			return ac.WriteSendBlock(batch, block)
		} else {
			// Receive block
			return ac.WriteReceiveBlock(batch, block)
		}
		return nil
	})
	//if (block.AccountAddress == nil) {
	//	return errors.New("Write block failed, because accountAddress is not exist")
	//}
	//
	//accountMeta := ac.accountStore.GetAccountMeta(block.AccountAddress)
	//GetBigIntBytesList
	//lastAccountBlockHeight := big.NewInt(-2)
	//
	//if accountMeta != nil {
	//	for _, token := range accountMeta.TokenList {
	//		if bytes.Equal(token.TokenId, block.TokenId) {
	//			lastAccountBlockHeight = token.LastAccountBlockHeight
	//			break
	//		}
	//	}
	//}
	//
	//fromHash := block.FromHash
	//
	//if fromHash == nil {
	//
	//	// It is send block
	//	if lastAccountBlockHeight.Cmp(big.NewInt(-2)) == 0 {
	//		return errors.New("Write send block failed, because the account does not have this token")
	//	}
	//
	//	lastAccountBlock, err := ac.GetBlockByHeight(lastAccountBlockHeight)
	//
	//	if err != nil {
	//		return errors.New(fmt.Sprintln("Write send block failed, because GetBlockByHeight failed. Error is ", err))
	//	}
	//
	//	if lastAccountBlock == nil || block.Amount.Cmp(lastAccountBlock.Balance) > 0 {
	//		return errors.New("Write send block failed, because the balance is not enough")
	//	}
	//} else {
	//	// It is receive block
	//	fromBlockMeta, err:= ac.GetBlockMeta(block.FromHash)
	//
	//	if fromBlockMeta == nil {
	//		return errors.New("Write receive block failed, because the from block is not exist")
	//	}
	//
	//	if err != nil {
	//		return errors.New(fmt.Sprintln("Write receive block failed, because GetBlockByHeight failed. Error is ", err))
	//	}
	//
	//
	//	accountId := big.NewInt(123)
	//
	//	if lastAccountBlockHeight.Cmp(big.NewInt(-2)) == 0 {
	//		// Write account meta
	//		if accountMeta == nil {
	//			accountMeta = &ledger.AccountMeta{
	//				AccountId: accountId,
	//			}
	//		}
	//
	//		accountMeta.TokenList = append(accountMeta.TokenList, &ledger.AccountSimpleToken{
	//			TokenId: []byte{1, 2, 3},
	//			LastAccountBlockHeight: big.NewInt(-1),
	//		})
	//
	//		ac.WriteAccountMeta(block.AccountAddress, accountMeta)
	//		ac.WriteAccountIdIndex(accountId, block.AccountAddress)
	//	}
	//
	//	// Write from block meta
	//	fromBlockMeta.Status = 2 // Closed
	//	//ac.WriteBlockMeta(block.FromHash, fromBlockMeta)
	//
	//}
	//
	//// 模拟key, 需要改
	//key :=  []byte("test")
	//
	//// Block serialize by protocol buffer
	//data, err := block.DbSerialize()
	//
	//if err != nil {
	//	return errors.New(fmt.Sprintln("Write send block failed. Error is ", err))
	//}
	//
	//ac.db.Leveldb.Put(key, data, nil)
	//
	//return nil
}

func (ac *AccountChain) WriteBlockBody (batch *leveldb.Batch, block *ledger.AccountBlock) error {
	return batchWrite(batch, ac.db.Leveldb, func(context *batchContext) error {
		return nil
	})
}

func (ac *AccountChain) WriteMintageBlock (batch *leveldb.Batch, block *ledger.AccountBlock) error {
	return batchWrite(batch, ac.db.Leveldb, func(context *batchContext) error {

		// Write block body
		if err := ac.WriteBlockBody(batch, block); err != nil{
			return err
		}

		// Write block meta
		if err := ac.WriteBlockMeta(batch, block.Hash, &ledger.AccountBlockMeta{}); err != nil{
			return err
		}


		//testTokenId := []byte("testTokenId")

		// Write TokenId Index
		//if err := ac.tokenStore.WriteTokenIdIndex(batch, testTokenId, big.NewInt(111), block.Hash); err != nil{
		//	return err
		//}
		//
		//// Write TokenName body
		//if err := ac.tokenStore.WriteTokenNameIndex(batch, "testTokenName", testTokenId); err != nil{
		//	return err
		//}
		//
		//
		//// Write TokenSymbol body
		//if err := ac.tokenStore.WriteTokenSymbolIndex(batch, "testTokenSymbol", testTokenId); err != nil{
		//	return err
		//}
		return nil
	})
}

func (ac * AccountChain) WriteSendBlock (batch *leveldb.Batch, block *ledger.AccountBlock) error {
	return batchWrite(batch, ac.db.Leveldb, func(context *batchContext) error {
		//accountMeta := ac.accountStore.GetAccountMeta(block.AccountAddress)
		//if accountMeta == nil {
		//	return errors.New("Write send block failed, because account is not exist")
		//}

		if bytes.Equal(block.To, []byte{0}) {
			// Mintage block
			return ac.WriteMintageBlock(batch, block)
		}

		return nil
	})
}

func (ac * AccountChain) WriteReceiveBlock (batch *leveldb.Batch, block *ledger.AccountBlock) error {
	return batchWrite(batch, ac.db.Leveldb, func(context *batchContext) error {
		return nil
	})
}

//func (ac * AccountChain) WriteAccountMeta (accountAddress []byte, accountMeta *ledger.AccountMeta) error {
//
//	return nil
//}
//
//func (ac * AccountChain) WriteAccountIdIndex (accountId *big.Int, accountAddress []byte) error {
//
//	return nil
//}

func (ac * AccountChain) WriteBlockMeta (batch *leveldb.Batch, accountBlockHash []byte, accountBlockMeta *ledger.AccountBlockMeta) error {
	return batchWrite(batch, ac.db.Leveldb, func(context *batchContext) error {
		return nil
	})
}


func (ac * AccountChain) GetBlockByHash (blockHash []byte) (*ledger.AccountBlock, error) {
	reader := ac.db.Leveldb

	block, err := reader.Get(blockHash, nil)
	if err != nil {
		return nil, err
	}
	accountBlock := &ledger.AccountBlock{}
	accountBlock.DbDeserialize(block)

	return accountBlock, nil
}

func (ac *AccountChain) GetLatestBlockHeightByAccountId (accountId *big.Int) (* big.Int, error){
	return nil, nil
}

func (ac *AccountChain) GetBlockListByAccountMeta (index int, num int, count int, meta *ledger.AccountMeta) ([]*ledger.AccountBlock, error) {
	latestBlockHeight, err := ac.GetLatestBlockHeightByAccountId(meta.AccountId)
	if err != nil {
		return nil, nil
	}
	startIndex := latestBlockHeight.Sub(latestBlockHeight, big.NewInt(int64(index * count)))
	key := createKey(DBKP_ACCOUNTBLOCK, meta.AccountId, startIndex)

	iter := ac.db.Leveldb.NewIterator(&util.Range{Start: key}, nil)
	defer iter.Release()

	var blockList []*ledger.AccountBlock
	for i:=0; i < num * count; i ++ {
		if iter.Prev() {
			break
		}

		if err := iter.Error(); err != nil {
			return nil, err
		}
		block := &ledger.AccountBlock{}

		err := block.DbDeserialize(iter.Value())
		if err != nil {
			return nil, err
		}

		blockList = append(blockList, block)
	}


	return blockList, nil
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