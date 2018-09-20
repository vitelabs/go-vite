package model

import (
	"bytes"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"path/filepath"
)

type UAccess struct {
	Chain *chain.Chain
	store *UnconfirmedSet
	log   log15.Logger
}

func NewUAccess(chain *chain.Chain, dataDir string) *UAccess {
	uAccess := &UAccess{
		Chain: chain,
		log:   log15.New("w", "uAccess"),
	}
	dbDir := filepath.Join(dataDir, "Chain")
	db, err := leveldb.OpenFile(dbDir, nil)
	if err != nil {
		uAccess.log.Error("ChainDb not find or create DB failed")
	}
	uAccess.store = NewUnconfirmedSet(db)
	return uAccess
}

func (access *UAccess) GetContractAddrListByGid(gid *types.Gid) (addrList []*types.Address, err error) {
	return access.store.GetContractAddrList(gid)
}

func (access *UAccess) WriteContractAddrToGid(batch *leveldb.Batch, gid types.Gid, address types.Address) error {
	var addrList []*types.Address
	var err error

	addrList, err = access.GetContractAddrListByGid(&gid)
	if addrList == nil && err != nil {
		access.log.Error("GetMeta", "error", err)
		return err
	} else {
		addrList = append(addrList, &address)
		return access.store.WriteGidAddrList(batch, &gid, addrList)
	}
}

func (access *UAccess) DeleteContractAddrFromGid(batch *leveldb.Batch, gid types.Gid, address types.Address) error {
	var addrList []*types.Address
	var err error

	addrList, err = access.GetContractAddrListByGid(&gid)
	if addrList == nil || err != nil {
		access.log.Error("GetContractAddrListByGid", "error", err)
		return err
	} else {
		for k, v := range addrList {
			if bytes.Equal(v.Bytes(), address.Bytes()) {
				addrList = append(addrList[0:k-1], addrList[k+1:]...)
				break
			}
		}
		return access.store.WriteGidAddrList(batch, &gid, addrList)
	}
}

func (access *UAccess) writeUnconfirmedMeta(batch *leveldb.Batch, block *ledger.AccountBlock) (err error) {
	addr := &block.ToAddress
	hash := &block.FromBlockHash

	value, err := access.store.GetMeta(addr, hash)
	if value == nil {
		// the commonSend add func
		if err != nil {
			access.log.Error("GetMeta", "error", err)
			return err
		}
		return access.store.WriteMeta(batch, addr, hash, 0)
	} else {
		// add the revert recvblock's sendBlock
		count := uint8(value[0])
		if count > 0 {
			count--
			return access.store.WriteMeta(batch, addr, hash, count)
		}
		return nil
	}
}

// only to delete the exist sendBlock, including the received-incorrect block and th correct block
func (access *UAccess) deleteUnconfirmedMeta(batch *leveldb.Batch, block *ledger.AccountBlock) (err error) {
	addr := &block.ToAddress
	hash := &block.FromBlockHash

	value, err := access.store.GetMeta(addr, hash)
	if value == nil {
		access.log.Error("can't find the corresponding sendBlock")
		return err
	}

	if err = access.store.DeleteMeta(batch, &block.ToAddress, &block.Hash); err != nil {
		access.log.Error("DeleteMeta", "error", err)
	}
	count := uint8(value[0])
	if count >= 3 {
		if err := access.store.DeleteMeta(batch, addr, hash); err != nil {
			return err
		}
		return nil
	} else {
		count++
		return access.store.WriteMeta(batch, addr, hash, count)
	}
	return nil
}

func (access *UAccess) GetUnconfirmedHashs(index, num, count uint64, addr *types.Address) ([]*types.Hash, error) {
	totalCount := (index + num) * count
	maxCount, err := access.store.GetCountByAddress(addr)
	if err != nil && err != leveldb.ErrNotFound {
		access.log.Error("GetUnconfirmedHashs", "error", err)
		return nil, err
	}
	if totalCount > maxCount {
		totalCount = maxCount
	}

	hashList, err := access.store.GetHashsByCount(totalCount, addr)
	if err != nil {
		access.log.Error("GetHashsByCount", "error", err)
		return nil, err
	}

	return hashList, nil
}

func (access *UAccess) GetUnconfirmedBlocks(index, num, count uint64, addr *types.Address) (blockList []*ledger.AccountBlock, err error) {
	hashList, err := access.GetUnconfirmedHashs(index, num, count, addr)
	if err != nil {
		return nil, err
	}
	for _, v := range hashList {
		block, err := access.Chain.GetAccountBlockByHash(v)
		if err != nil || block == nil {
			access.log.Error("ContractWorker.GetBlockByHash", "error", err)
			continue
		}
		blockList = append(blockList, block)
	}
	return nil, nil
}

func (access *UAccess) GetCommonAccInfo(addr *types.Address) (info *CommonAccountInfo, err error) {
	infoMap, number, err := access.GetCommonAccTokenInfoMap(addr)
	if err != nil {
		return nil, err
	}
	info = &CommonAccountInfo{
		AccountAddress:      addr,
		TotalNumber:         number,
		TokenBalanceInfoMap: infoMap,
	}

	return info, nil
}

func (access *UAccess) GetAllUnconfirmedBlocks(addr types.Address) (blockList []*ledger.AccountBlock, err error) {
	hashList, err := access.store.GetHashList(&addr)
	if err != nil {
		return nil, err
	}
	result := make([]*ledger.AccountBlock, len(hashList))

	for i, v := range hashList {
		block, err := access.Chain.GetAccountBlockByHash(v)
		if err != nil || block == nil {
			access.log.Error("ContractWorker.GetBlockByHash", "error", err)
			continue
		}
		result[i] = block
	}

	return result, nil
}

func (access *UAccess) GetCommonAccTokenInfoMap(addr *types.Address) (map[types.TokenTypeId]*TokenBalanceInfo, uint64, error) {
	infoMap := make(map[types.TokenTypeId]*TokenBalanceInfo)
	hashList, err := access.store.GetHashList(addr)
	if err != nil {
		access.log.Error("GetCommonAccTokenInfoMap.GetHashList", "error", err)
		return nil, 0, err
	}
	for _, v := range hashList {
		block, err := access.Chain.GetAccountBlockByHash(v)
		if err != nil || block == nil {
			access.log.Error("ContractWorker.GetBlockByHash", "error", err)
			continue
		}
		ti, ok := infoMap[block.TokenId]
		if ok {
			ti.Number += 1
			ti.TotalAmount.Add(&ti.TotalAmount, block.Amount)
		} else {
			token, err := access.Chain.GetTokenInfoById(&block.TokenId)
			if err != nil {
				access.log.Error("func GetUnconfirmedAccount.GetByTokenId failed", "error", err)
				return nil, 0, err
			}
			infoMap[block.TokenId].Token = *token
			infoMap[block.TokenId].TotalAmount = *block.Amount
			infoMap[block.TokenId].Number = 1
		}

	}
	return infoMap, uint64(len(hashList)), err
}

func (access *UAccess) GetAccountQuota(addr types.Address, hashes types.Hash) uint64 {
	return 0
}

func (access *UAccess) GetReceiveTimes(addr *types.Address, hash *types.Hash) (uint8, error) {
	value, err := access.store.GetMeta(addr, hash)
	if err != nil {
		// include find nil and db errs
		access.log.Error("GetMeta", "error", err)
		return 0, err
	}
	return uint8(value[0]), nil
}
