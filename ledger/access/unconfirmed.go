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
	"bytes"
)

type unconfirmListener map[string] chan int
type UnconfirmedAccess struct {
	store                *vitedb.Unconfirmed
	writeNewAccountMutex sync.Mutex
	listener             *unconfirmListener
}

var unconfirmedAccess = &UnconfirmedAccess{
	store:                vitedb.GetUnconfirmed(),
	writeNewAccountMutex: sync.Mutex{},
	listener:             &unconfirmListener{},
}

func GetUnconfirmedAccess() *UnconfirmedAccess {
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

func (ucfa *UnconfirmedAccess) GetUnconfirmedHashs(index int, num int, count int, accountId *big.Int, tokenId *types.TokenTypeId) ([]*types.Hash, error) {
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

func (ucfa *UnconfirmedAccess) GetUnconfirmedAccountMeta(addr *types.Address) (*ledger.UnconfirmedMeta, error) {
	return ucfa.store.GetUnconfirmedMeta(addr)
}

func (ucfa *UnconfirmedAccess) GetAccountHashList(accountId *big.Int, tokenId *types.TokenTypeId) ([]*types.Hash, error) {
	return ucfa.store.GetUnconfirmedHashList(accountId, tokenId)
}

func (ucfa *UnconfirmedAccess) WriteBlock(batch *leveldb.Batch, addr *types.Address, block *ledger.AccountBlock) error {
	// judge whether the block exists
	//block, err := accountChainAccess.GetBlockByHash(hash)
	//if err != nil {
	//	return &AcWriteError{
	//		Code: WacDefaultErr,
	//		Err:  errors.New("Write unconfirmed failed, because getting the block by hash failed. Error is " + err.Error()),
	//	}
	//}

	// judge whether the address exists
	uAccMeta, err := ucfa.store.GetUnconfirmedMeta(addr)
	if err != nil && err != leveldb.ErrNotFound {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}
	if uAccMeta == nil {
		ucfa.writeNewAccountMutex.Lock()
		defer ucfa.writeNewAccountMutex.Unlock()

		uAccMeta, err = ucfa.CreateNewUcfmMeta(addr, block)
		if err != nil {
			return &AcWriteError{
				Code: WacDefaultErr,
				Err:  err,
			}
		}
	} else {
		// Upodate total number of this account's unconfirmedblocks
		var number = &big.Int{}
		uAccMeta.TotalNumber = number.Add(uAccMeta.TotalNumber, big.NewInt(1))

		var tokenExist = false
		// Update the total amount of the unconfirmed info per token
		for index, tokeInfo := range uAccMeta.TokenInfoList {
			if tokeInfo.TokenId == block.TokenId {
				var amount = &big.Int{}
				uAccMeta.TokenInfoList[index].TotalAmount = amount.Add(tokeInfo.TotalAmount, block.Amount)
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
	if err != nil && err != leveldb.ErrNotFound {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}
	hashList = append(hashList, block.Hash)

	if err := ucfa.store.WriteMeta(batch, addr, uAccMeta); err != nil {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}
	if err := ucfa.store.WriteHashList(batch, uAccMeta.AccountId, block.TokenId, hashList); err != nil {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}

	// Add to the Listener
	ucfa.SendSignalToListener(addr)
	return nil
}

func (ucfa *UnconfirmedAccess) CreateNewUcfmMeta(addr *types.Address, block *ledger.AccountBlock) (*ledger.UnconfirmedMeta, error) {
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
	accountMeta := &ledger.UnconfirmedMeta{
		AccountId:     accMeta.AccountId,
		TokenInfoList: []*ledger.TokenInfo{ti},
		TotalNumber:   big.NewInt(1),
	}
	return accountMeta, nil
}

func (ucfa *UnconfirmedAccess) DeleteBlock(batch *leveldb.Batch, addr *types.Address, block *ledger.AccountBlock) error {
	uAccMeta, err := ucfa.store.GetUnconfirmedMeta(addr)
	if err != nil && err != leveldb.ErrNotFound {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}
	if uAccMeta == nil {
		err := ucfa.store.DeleteMeta(batch, addr)
		ucfa.RemoveListener(addr)
		return errors.New("Delete unconfirmed failed, because uAccMeta is empty. Log:" + err.Error())
	}

	hashList, err := ucfa.store.GetUnconfirmedHashList(uAccMeta.AccountId, block.TokenId)
	if err != nil && err != leveldb.ErrNotFound {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}
	if hashList == nil {
		err := ucfa.store.DeleteHashList(batch, uAccMeta.AccountId, block.TokenId)
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  errors.New("Delete unconfirmed hashList failed, because hashList is empty. Log:" + err.Error()),
		}
	}

	ucfa.writeNewAccountMutex.Lock()
	defer ucfa.writeNewAccountMutex.Unlock()

	// Update the TotalNumber of UnconfirmedMeta
	var number = &big.Int{}
	number.Sub(uAccMeta.TotalNumber, big.NewInt(1))
	uAccMeta.TotalNumber = number
	if number == big.NewInt(0) {
		// Remove the Listener.
		// But remain the addr:meta value in db, although the meta has no specific value.
		ucfa.RemoveListener(addr)
	}

	// Update the TotalAmount of the TokenInfo
	for index, tokeInfo := range uAccMeta.TokenInfoList {
		if tokeInfo.TokenId == block.TokenId {
			var amount = &big.Int{}
			amount.Sub(tokeInfo.TotalAmount, block.Amount)
			uAccMeta.TokenInfoList[index].TotalAmount = amount
			break
		}
	}

	// Remove the hash from the HashList
	for index, data := range hashList {
		if bytes.Equal(data.Bytes(), block.Hash.Bytes()) {
			hashList = append(hashList[:index], hashList[index+1:]...)
		}
	}

	// if HashList is empty,
	// Delete key-value of the HashList and remove the TokenInfo from the TokenInfoList.
	if len(hashList) <= 0 {
		if err := ucfa.store.DeleteHashList(batch, uAccMeta.AccountId, block.TokenId); err != nil {
			return &AcWriteError{
				Code: WacDefaultErr,
				Err:  err,
			}
		}
		for index, tokeInfo := range uAccMeta.TokenInfoList {
			if tokeInfo.TokenId == block.TokenId {
				uAccMeta.TokenInfoList = append(uAccMeta.TokenInfoList[:index], uAccMeta.TokenInfoList[index+1:]...)
			}
		}
	}

	if err := ucfa.store.WriteMeta(batch, addr, uAccMeta); err != nil {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}
	if err := ucfa.store.WriteHashList(batch, uAccMeta.AccountId, block.TokenId, hashList); err != nil {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}
	return nil
}

func (ucfa *UnconfirmedAccess) SendSignalToListener (addr *types.Address) {
	(*ucfa.listener)[addr.String()] <- 1
}

func (ucfa *UnconfirmedAccess) RemoveListener (addr *types.Address) {
	delete(*ucfa.listener, addr.String())
}

func (ucfa *UnconfirmedAccess) GetListener (addr *types.Address) chan int {
	signal, _ := (*ucfa.listener)[addr.String()]
	return signal
}