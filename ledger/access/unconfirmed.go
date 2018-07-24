package access

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log"
	"github.com/vitelabs/go-vite/vitedb"
	"math/big"
	"sync"
)

type unconfirmedListener map[types.Address]chan<- struct{}

var unconfirmedAccess = &UnconfirmedAccess{
	store:             vitedb.GetUnconfirmed(),
	writeAccountMutex: sync.Mutex{},
	uwMutex:           &ucfmWriteMutex{},
	listener:          &unconfirmedListener{},
}

type UnconfirmedAccess struct {
	store             *vitedb.Unconfirmed
	writeAccountMutex sync.Mutex
	uwMutex           *ucfmWriteMutex
	listener          *unconfirmedListener
}

func GetUnconfirmedAccess() *UnconfirmedAccess {
	return unconfirmedAccess
}

type ucfmWriteMutex map[types.Address]*ucfmWriteMuteBody
type ucfmWriteMuteBody struct {
	Reference bool
}

var uWMMutex sync.Mutex

func (uwm *ucfmWriteMutex) Lock(block *ledger.AccountBlock) *AcWriteError {
	uWMMutex.Lock()
	defer uWMMutex.Unlock()
	uwmBody, ok := (*uwm)[*block.AccountAddress]

	if !ok || uwmBody == nil {
		uwmBody = &ucfmWriteMuteBody{
			Reference: false,
		}
		(*uwm)[*block.AccountAddress] = uwmBody
	}

	if uwmBody.Reference {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  errors.New("Lock failed"),
		}
	}

	uwmBody.Reference = true
	return nil
}

func (uwm *ucfmWriteMutex) UnLock(block *ledger.AccountBlock) {
	uWMMutex.Lock()
	defer uWMMutex.Unlock()

	uwmBody, ok := (*uwm)[*block.AccountAddress]
	if !ok {
		return
	}
	uwmBody.Reference = false
}

func (ucfa *UnconfirmedAccess) GetUnconfirmedHashsByTkId(index, num, count int, addr *types.Address, tokenId *types.TokenTypeId) ([]*types.Hash, error) {
	acMeta, err := accountAccess.GetAccountMeta(addr)
	if err != nil {
		return nil, err
	}

	var hList []*types.Hash

	hashList, err := ucfa.store.GetAccHashListByTkId(acMeta.AccountId, tokenId)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}
	if hashList == nil {
		return hList, nil
	}

	log.Info("GetUnconfirmedBlock/GetAccountHashList:len(hashList)=", len(hashList))
	for i := index * count; i < (index+num)*count && i < len(hashList); i++ {
		hash := hashList[i]
		hList = append(hList, hash)
	}
	return hList, nil
}

func (ucfa *UnconfirmedAccess) GetUnconfirmedHashs(number int, addr *types.Address) ([]*types.Hash, error) {
	meta, err := ucfa.store.GetUnconfirmedMeta(addr)
	if err != nil {
		return nil, err
	}
	numberInt := big.NewInt(int64(number))
	if numberInt.Cmp(meta.TotalNumber) == 1 || big.NewInt(0).Cmp(meta.TotalNumber) == 0{
		return nil, errors.New("The number to get is out of range.")
	}
	hashList, err := ucfa.store.GetAccTotalHashList(meta.AccountId)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}
	return hashList[0:number], nil
}

func (ucfa *UnconfirmedAccess) GetUnconfirmedAccountMeta(addr *types.Address) (*ledger.UnconfirmedMeta, error) {
	return ucfa.store.GetUnconfirmedMeta(addr)
}

func (ucfa *UnconfirmedAccess) WriteBlock(batch *leveldb.Batch, block *ledger.AccountBlock) error {
	// judge whether the block exists
	//block, err := accountChainAccess.GetBlockByHash(hash)
	//if err != nil {
	//	return &AcWriteError{
	//		Code: WacDefaultErr,
	//		Err:  errors.New("Write unconfirmed failed, because getting the block by hash failed. Error is " + err.Error()),
	//	}
	//}

	// judge whether the address exists
	uAccMeta, err := ucfa.store.GetUnconfirmedMeta(block.AccountAddress)
	if err != nil && err != leveldb.ErrNotFound {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}
	if err := ucfa.uwMutex.Lock(block); err != nil {
		return err
	}
	defer ucfa.uwMutex.UnLock(block)

	if uAccMeta != nil {
		// Upodate total number of this account's unconfirmedblocks
		var number = &big.Int{}
		uAccMeta.TotalNumber = number.Add(uAccMeta.TotalNumber, big.NewInt(1))

		var tokenExist = false
		// Update the total amount of the unconfirmed info per token
		for index, tokeInfo := range uAccMeta.TokenInfoList {
			if bytes.Equal(tokeInfo.TokenId.Bytes(), block.TokenId.Bytes()) {
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
	} else {
		ucfa.writeAccountMutex.Lock()
		defer ucfa.writeAccountMutex.Unlock()

		uAccMeta, err = ucfa.CreateNewUcfmMeta(block)
		if err != nil {
			return &AcWriteError{
				Code: WacDefaultErr,
				Err:  err,
			}
		}
	}
	hashList, err := ucfa.store.GetAccHashListByTkId(uAccMeta.AccountId, block.TokenId)
	if err != nil && err != leveldb.ErrNotFound {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}
	hashList = append(hashList, block.Hash)

	if err := ucfa.store.WriteMeta(batch, block.AccountAddress, uAccMeta); err != nil {
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
	_, ok := (*ucfa.listener)[*block.AccountAddress]
	if ok {
		ucfa.SendSignalToListener(*block.AccountAddress)
	}
	return nil
}

func (ucfa *UnconfirmedAccess) CreateNewUcfmMeta(block *ledger.AccountBlock) (*ledger.UnconfirmedMeta, error) {
	// Get the accountId
	accMeta, err := accountAccess.GetAccountMeta(block.AccountAddress)
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

func (ucfa *UnconfirmedAccess) DeleteBlock(batch *leveldb.Batch, block *ledger.AccountBlock) error {

	uAccMeta, err := ucfa.store.GetUnconfirmedMeta(block.AccountAddress)
	if err != nil && err != leveldb.ErrNotFound {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}
	if uAccMeta == nil {
		ucfa.writeAccountMutex.Lock()
		defer ucfa.writeAccountMutex.Unlock()

		err := ucfa.store.DeleteMeta(batch, block.AccountAddress)
		return errors.New("Delete unconfirmed failed, because uAccMeta is empty. Log:" + err.Error())
	}

	hashList, err := ucfa.store.GetAccHashListByTkId(uAccMeta.AccountId, block.TokenId)
	if err != nil && err != leveldb.ErrNotFound {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}
	if err := ucfa.uwMutex.Lock(block); err != nil {
		return err
	}
	defer ucfa.uwMutex.UnLock(block)

	if hashList == nil {
		err := ucfa.store.DeleteHashList(batch, uAccMeta.AccountId, block.TokenId)
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  errors.New("Delete unconfirmed hashList failed, because hashList is empty. Log:" + err.Error()),
		}
	}

	// Update the TotalNumber of UnconfirmedMeta
	var number = &big.Int{}
	number.Sub(uAccMeta.TotalNumber, big.NewInt(1))
	uAccMeta.TotalNumber = number

	// Update the TotalAmount of the TokenInfo
	for index, tokeInfo := range uAccMeta.TokenInfoList {
		if bytes.Equal(tokeInfo.TokenId.Bytes(), block.TokenId.Bytes()) {
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
			if bytes.Equal(tokeInfo.TokenId.Bytes(), block.TokenId.Bytes()) {
				uAccMeta.TokenInfoList = append(uAccMeta.TokenInfoList[:index], uAccMeta.TokenInfoList[index+1:]...)
			}
		}
	}

	if err := ucfa.store.WriteMeta(batch, block.AccountAddress, uAccMeta); err != nil {
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

var listenerMutex sync.Mutex

func (ucfa *UnconfirmedAccess) SendSignalToListener(addr types.Address) {
	listenerMutex.Lock()
	(*ucfa.listener)[addr] <- struct{}{}
	listenerMutex.Unlock()
}

func (ucfa *UnconfirmedAccess) RemoveListener(addr types.Address) {
	listenerMutex.Lock()
	defer listenerMutex.Unlock()
	delete(*ucfa.listener, addr)

}

func (ucfa *UnconfirmedAccess) AddListener(addr types.Address, change chan<- struct{}) {
	listenerMutex.Lock()
	defer listenerMutex.Unlock()
	(*ucfa.listener)[addr] = change
}

func (ucfa *UnconfirmedAccess) UnconfirmedCallBack(batch *leveldb.Batch, block *ledger.AccountBlock) error {
	// meta.status: 1 means open, 2 means closed
	if block.Meta.Status == 2 {
		if ucfa.HashUnconfirmedBool(block) {
			fromBlock, err := accountChainAccess.GetBlockByHash(block.FromHash)
			if err != nil {
				return err
			}
			ucfa.WriteBlock(batch, fromBlock)
		}
	}
	if err := ucfa.DeleteBlock(batch, block); err != nil {
		return err
	}
	return nil
}

func (ucfa *UnconfirmedAccess) HashUnconfirmedBool(block *ledger.AccountBlock) bool {
	hashList, err := ucfa.store.GetAccHashListByTkId(block.Meta.AccountId, block.TokenId)
	if err != nil {
		return false
	}
	for _, hash := range hashList {
		if bytes.Equal(hash.Bytes(), block.FromHash.Bytes()) {
			return true
		}
	}
	return false
}
