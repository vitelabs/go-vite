package access

import (
	"bytes"
	"github.com/vitelabs/go-vite/log15"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vitedb"
	"math/big"
	"sync"
)

var uLog = log15.New("module", "ledger/access/unconfirmed")

var ucListener = make(map[types.Address]chan<- struct{})

var unconfirmedAccess *UnconfirmedAccess

type UnconfirmedAccess struct {
	store   *vitedb.Unconfirmed
	uwMutex *ucfmWriteMutex
}

func GetUnconfirmedAccess() *UnconfirmedAccess {
	if unconfirmedAccess == nil {
		unconfirmedAccess = &UnconfirmedAccess{
			store:   vitedb.GetUnconfirmed(),
			uwMutex: &ucfmWriteMutex{},
		}

	}
	return unconfirmedAccess
}

type ucfmWriteMutex map[types.Address]*ucfmWriteMuteBody
type ucfmWriteMuteBody struct {
	writeLock sync.Mutex
}

var uWMMutex sync.Mutex

func (uwm *ucfmWriteMutex) Lock(block *ledger.AccountBlock) {
	uWMMutex.Lock()
	uwmBody, ok := (*uwm)[*block.To]

	if !ok || uwmBody == nil {
		uwmBody = &ucfmWriteMuteBody{}
		(*uwm)[*block.To] = uwmBody
	}
	uWMMutex.Unlock()

	uwmBody.writeLock.Lock()
}

func (uwm *ucfmWriteMutex) UnLock(block *ledger.AccountBlock) {
	uwmBody, ok := (*uwm)[*block.To]
	if !ok {
		return
	}
	uwmBody.writeLock.Unlock()
}

func (ucfa *UnconfirmedAccess) GetUnconfirmedHashsByTkId(index, num, count int, addr *types.Address, tokenId *types.TokenTypeId) ([]*types.Hash, error) {
	var hList []*types.Hash

	hashList, err := ucfa.store.GetAccHashListByTkId(addr, tokenId)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}
	if hashList == nil {
		return hList, nil
	}

	uLog.Info("GetUnconfirmedBlock/GetAccountHashList", "len(hashList)", len(hashList))
	for i := index * count; i < (index+num)*count && i < len(hashList); i++ {
		hash := hashList[i]
		hList = append(hList, hash)
	}
	return hList, nil
}

func (ucfa *UnconfirmedAccess) GetUnconfirmedHashs(index, num, count int, addr *types.Address) ([]*types.Hash, error) {
	meta, err := ucfa.store.GetUnconfirmedMeta(addr)
	if err != nil {
		return nil, err
	}
	numberInt := big.NewInt(int64((index + num) * count))
	if big.NewInt(0).Cmp(meta.TotalNumber) == 0 {
		return nil, nil
	}
	hashList, err := ucfa.store.GetAccTotalHashList(addr)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}
	if numberInt.Cmp(meta.TotalNumber) == 1 {
		return hashList[index*count:], nil
	}
	return hashList[index*count : (index+num)*count], nil
}

func (ucfa *UnconfirmedAccess) GetUnconfirmedAccountMeta(addr *types.Address) (*ledger.UnconfirmedMeta, error) {
	return ucfa.store.GetUnconfirmedMeta(addr)
}

func (ucfa *UnconfirmedAccess) WriteBlock(batch *leveldb.Batch, block *ledger.AccountBlock) error {
	if block.To == nil || block.TokenId == nil {
		err := errors.New("send_block's value is invalid")
		uLog.Info("Unconfirmed", "err", err)
		return err
	}

	// judge whether the address exists
	uAccMeta, err := ucfa.store.GetUnconfirmedMeta(block.To)
	if err != nil && err != leveldb.ErrNotFound {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}
	ucfa.uwMutex.Lock(block)
	defer ucfa.uwMutex.UnLock(block)

	if uAccMeta != nil {
		// [tmp] Check data.
		//uLog.Info("Unconfirmed: before write:")
		//uLog.Info("UnconfirmedMeta: AccountId:", uAccMeta.AccountId, ", TotalNumber:", uAccMeta.TotalNumber, ", TokenInfoList:")
		//for idx, tokenInfo := range uAccMeta.TokenInfoList {
		//	uLog.Info("TokenInfo", idx, ":", tokenInfo.TokenId, tokenInfo.TotalAmount)
		//}

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
		uAccMeta, err = ucfa.CreateNewUcfmMeta(block)
		if err != nil {
			return &AcWriteError{
				Code: WacDefaultErr,
				Err:  err,
			}
		}
	}
	hashList, err := ucfa.store.GetAccHashListByTkId(block.To, block.TokenId)
	if err != nil && err != leveldb.ErrNotFound {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}
	hashList = append(hashList, block.Hash)

	if err := ucfa.store.WriteMeta(batch, block.To, uAccMeta); err != nil {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}
	if err := ucfa.store.WriteHashList(batch, block.To, block.TokenId, hashList); err != nil {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}

	return nil
}

func (ucfa *UnconfirmedAccess) CreateNewUcfmMeta(block *ledger.AccountBlock) (*ledger.UnconfirmedMeta, error) {
	ti := &ledger.TokenInfo{
		TotalAmount: block.Amount,
		TokenId:     block.TokenId,
	}
	// Create account meta which will be write to database later
	accountMeta := &ledger.UnconfirmedMeta{
		TokenInfoList: []*ledger.TokenInfo{ti},
		TotalNumber:   big.NewInt(1),
	}
	return accountMeta, nil
}

func (ucfa *UnconfirmedAccess) DeleteBlock(batch *leveldb.Batch, block *ledger.AccountBlock) error {
	if block.To == nil || block.TokenId == nil {
		err := errors.New("send_block's value is invalid")
		uLog.Info("Unconfirmed", "err", err)
		return err
	}

	uAccMeta, err := ucfa.store.GetUnconfirmedMeta(block.To)
	if err != nil && err != leveldb.ErrNotFound {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}

	ucfa.uwMutex.Lock(block)
	defer ucfa.uwMutex.UnLock(block)

	if uAccMeta == nil {
		err := ucfa.store.DeleteMeta(batch, block.To)
		return errors.New("delete unconfirmed failed, because uAccMeta is empty, error:" + err.Error())
	}

	//// [tmp] Check data.
	//uLog.Info("Unconfirmed: before delete:")
	//uLog.Info("UnconfirmedMeta: AccountId:", uAccMeta.AccountId, ", TotalNumber:", uAccMeta.TotalNumber, ", TokenInfoList:")
	//for idx, tokenInfo := range uAccMeta.TokenInfoList {
	//	uLog.Info("TokenInfo", idx, ":", tokenInfo.TokenId, tokenInfo.TotalAmount)
	//}

	hashList, err := ucfa.store.GetAccHashListByTkId(block.To, block.TokenId)
	if err != nil && err != leveldb.ErrNotFound {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}

	if hashList == nil {
		err := ucfa.store.DeleteHashList(batch, block.To, block.TokenId)
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  errors.New("delete unconfirmed hashList failed, because hashList is empty, error: " + err.Error()),
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
		if err := ucfa.store.DeleteHashList(batch, block.To, block.TokenId); err != nil {
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

	if err := ucfa.store.WriteMeta(batch, block.To, uAccMeta); err != nil {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}
	if err := ucfa.store.WriteHashList(batch, block.To, block.TokenId, hashList); err != nil {
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
	defer listenerMutex.Unlock()
	uLog.Info("UnconfirmedAccess.SendSignalToListener: try to send.")
	if targetChannel, ok := ucListener[addr]; ok {
		uLog.Info("UnconfirmedAccess.SendSignalToListener: start send signal.")
		targetChannel <- struct{}{}
	}
	uLog.Info("UnconfirmedAccess.SendSignalToListener: Send signal to listener success.")
}

func (ucfa *UnconfirmedAccess) RemoveListener(addr types.Address) {
	listenerMutex.Lock()
	defer listenerMutex.Unlock()
	delete(ucListener, addr)
	uLog.Info("Unconfirmed: Remove account's listener success.")
}

func (ucfa *UnconfirmedAccess) AddListener(addr types.Address, change chan<- struct{}) {
	listenerMutex.Lock()
	defer listenerMutex.Unlock()
	ucListener[addr] = change
	uLog.Info("AddListener", "AddListener", ucListener[addr], "change", change)
	uLog.Info("Unconfirmed: Add account's listener success.")
}

func (ucfa *UnconfirmedAccess) HashUnconfirmedBool(addr *types.Address, tkId *types.TokenTypeId, hash *types.Hash) bool {
	hashList, err := ucfa.store.GetAccHashListByTkId(addr, tkId)
	if err != nil {
		return false
	}
	for _, h := range hashList {
		if bytes.Equal(h.Bytes(), hash.Bytes()) {
			return true
		}
	}
	return false
}
