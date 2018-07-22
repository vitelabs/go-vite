package access

import (
	"github.com/vitelabs/go-vite/vitedb"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"github.com/vitelabs/go-vite/log"
	"github.com/syndtr/goleveldb/leveldb"
	"sync"
	"github.com/pkg/errors"
	"fmt"
)

type UnconfirmedAccess struct {
	store 			*vitedb.Unconfirmed
	writeNewAccountMutex sync.Mutex
}

var unconfirmedAccess = &UnconfirmedAccess {
	store: 		vitedb.GetUnconfirmed(),
	writeNewAccountMutex: sync.Mutex{},
}

func GetUnconfirmedAccess () *UnconfirmedAccess {
	return unconfirmedAccess
}

//func (ucfa *UnconfirmedAccess) GetUnconfirmedBlocks (index int, num int, count int, accountId *big.Int, tokenId *types.TokenTypeId) ([]*ledger.AccountBlock, error) {
//	var blockList []*ledger.AccountBlock
//
//	hashList, err := ucfa.GetAccountHashList(accountId, tokenId)
//	if err != nil {
//		return nil, err
//	}
//	log.Info("GetUnconfirmedBlock/GetAccountHashList:len(hashList)=", len(*hashList))
//
//	for i := 0; i < num*count; i++ {
//		hash := (*hashList)[index*count]
//		block, ghErr := accountChainAccess.GetBlockByHash(hash)
//		if ghErr != nil {
//			return nil, ghErr
//		}
//		blockList = append(blockList, block)
//	}
//	return blockList, nil
//}

func (ucfa *UnconfirmedAccess) GetUnconfirmedHashs (index int, num int, count int, accountId *big.Int, tokenId *types.TokenTypeId) ([]*types.Hash, error) {
	var hList []*types.Hash

	hashList, err := ucfa.GetAccountHashList(accountId, tokenId)
	if err != nil {
		return nil, err
	}
	log.Info("GetUnconfirmedBlock/GetAccountHashList:len(hashList)=", len(hashList))

	for i := 0; i < num*count; i++ {
		hash := hashList[index*count]
		hList = append(hList, hash)
	}
	return hList, nil
}

func (ucfa *UnconfirmedAccess) GetUnconfirmedAccountMeta (addr *types.Address) (*ledger.UnconfirmedMeta, error) {
	return ucfa.store.GetUnconfirmedMeta(addr)
}

func (ucfa *UnconfirmedAccess) GetAccountHashList (accountId *big.Int, tokenId *types.TokenTypeId) ([]*types.Hash, error) {
	return ucfa.store.GetUnconfirmedHashList(accountId, tokenId)
}

func (ucfa *UnconfirmedAccess) WriteBlock (batch *leveldb.Batch, addr *types.Address, hash *types.Hash) error {
	// judge whether the address exists
	uAccMeta, err := ucfa.store.GetUnconfirmedMeta(addr)
	if err != nil && err != leveldb.ErrNotFound{
		return &AcWriteError {
			Code: WacDefaultErr,
			Err: err,
		}
	}
	// judge whether the block exists
	block, err := accountChainAccess.GetBlockByHash(hash)
	if err != nil {
		return errors.New("Write unconfirmed failed, because getting the block by hash failed. Error is " + err.Error())
	}
	if uAccMeta == nil {
		uAccMeta, err = ucfa.CreateNewUcfmMeta(addr, block)
	} else {
		ucfa.writeNewAccountMutex.Lock()
		defer ucfa.writeNewAccountMutex.Unlock()

		// Upodate total number of this account's unconfirmedblocks
		uAccMeta.TotalNumber.Add(uAccMeta.TotalNumber, big.NewInt(1))

		var tokenExist = false
		// Update the total amount of the unconfirmed info per token
		for index, tokeInfo := range uAccMeta.TokenInfoList {
			if tokeInfo.TokenId == block.TokenId {
				tokeInfo.TotalAmount.Add(tokeInfo.TotalAmount, block.Amount)
				uAccMeta.TokenInfoList[index].TotalAmount = tokeInfo.TotalAmount
				tokenExist = true
				break
			}
		}
		if !tokenExist {
			var tokenInfo = &ledger.TokenInfo{
				TokenId:     block.TokenId,
				TotalAmount: block.Amount,
			}
			uAccMeta.TokenInfoList = append(uAccMeta.TokenInfoList, tokenInfo)
		}
	}

	hashList, err := ucfa.store.GetUnconfirmedHashList(uAccMeta.AccountId, block.TokenId)
	if err != nil && err != leveldb.ErrNotFound{
		return &AcWriteError {
			Code: WacDefaultErr,
			Err: err,
		}
	}
	hashList = append(hashList, hash)

	if err := ucfa.store.WriteMeta(batch, addr, uAccMeta); err != nil {
		return &AcWriteError {
			Code: WacDefaultErr,
			Err: err,
		}
	}
	if err := ucfa.store.WriteHashList(batch, uAccMeta.AccountId, block.TokenId, hashList); err != nil {
		return &AcWriteError {
			Code: WacDefaultErr,
			Err: err,
		}
	}

	SendSignalToListener()

	return nil
}

func (ucfa *UnconfirmedAccess) CreateNewUcfmMeta (addr *types.Address, block *ledger.AccountBlock) (*ledger.UnconfirmedMeta, error) {
	// Get the accountId
	accMeta, err := accountAccess.GetAccountMeta(addr)
	if err != nil {
		return nil, errors.New("[CreateNewUcfmMeta.GetAccountMeta]：" + err.Error())
	}
	ti := &ledger.TokenInfo{
		TotalAmount: block.Amount,
		TokenId:     block.TokenId,
	}
	// Create account meta which will be write to database later
	accountMeta := &ledger.UnconfirmedMeta {
		AccountId: accMeta.AccountId,
		TokenInfoList: []*ledger.TokenInfo{ti},
		TotalNumber: big.NewInt(1),
	}
	return accountMeta, nil
}

func (ucfa *UnconfirmedAccess) DeleteBlock (batch *leveldb.Batch, addr *types.Address, hash *types.Hash) error {
	uAccMeta, err := ucfa.store.GetUnconfirmedMeta(addr)
	if err != nil && err != leveldb.ErrNotFound {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err: err,
		}
	}
	if uAccMeta == nil {
		return errors.New("Delete unconfirmed failed, because address doesn't exists. Error is " + err.Error())
	}

	block, err := accountChainAccess.GetBlockByHash(hash)
	if err != nil {
		return errors.New("Delete unconfirmed failed, because getting the block by hash failed. Error is " + err.Error())
	}

	hashList, err := ucfa.store.GetUnconfirmedHashList(uAccMeta.AccountId, block.TokenId)
	if err != nil && err != leveldb.ErrNotFound{
		return &AcWriteError {
			Code: WacDefaultErr,
			Err: err,
		}
	}
	if hashList == nil || len(hashList) <= 0 {
		return errors.New("Delete unconfirmed failed, because hash doesn't exists in the hashList. Error is " + err.Error())
	}

	ucfa.writeNewAccountMutex.Lock()
	defer ucfa.writeNewAccountMutex.Unlock()

	var number = &big.Int{}
	number.Sub(uAccMeta.TotalNumber, big.NewInt(1))

	for index, tokeInfo := range uAccMeta.TokenInfoList {
		if tokeInfo.TokenId == block.TokenId {
			tokeInfo.TotalAmount.Sub(tokeInfo.TotalAmount, block.Amount)
			uAccMeta.TokenInfoList[index].TotalAmount = tokeInfo.TotalAmount
			break
		}
	}

	for index, data := range hashList {
		if data == hash {
			hashList = append(hashList[:index], hashList[index+1:]...)
			fmt.Println(hashList)
		}
	}

	if err := ucfa.store.WriteMeta(batch, addr, uAccMeta); err != nil {
		return &AcWriteError {
			Code: WacDefaultErr,
			Err: err,
		}
	}
	if err := ucfa.store.WriteHashList(batch, uAccMeta.AccountId, block.TokenId, hashList); err != nil {
		return &AcWriteError {
			Code: WacDefaultErr,
			Err: err,
		}
	}
	return nil
}


func SendSignalToListener () () {

}


