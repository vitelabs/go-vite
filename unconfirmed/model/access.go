package model

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"path/filepath"
)

type UAccess struct {
	store *UnconfirmedSet
	chain *chain.Chain
	log   log15.Logger
}

func NewUAccess(chain *chain.Chain, dataDir string) *UAccess {
	uAccess := &UAccess{
		chain: chain,

		log: log15.New("w", "uAccess"),
	}
	dbDir := filepath.Join(dataDir, "chain")
	db, err := leveldb.OpenFile(dbDir, nil)
	if err != nil {
		uAccess.log.Error("ChainDb not find or create DB failed")
	}
	uAccess.store = NewUnconfirmedSet(db)
	return uAccess
}

func (access *UAccess) GetAddrListByGid(gid *types.Gid) (addrList []*types.Address, err error) {
	return nil, nil
}

func (access *UAccess) writeUnconfirmedMeta(batch *leveldb.Batch, block *ledger.AccountBlock) (err error) {
	if err = access.store.WriteMeta(batch, &block.ToAddress, &block.Hash); err != nil {
		access.log.Error("WriteMeta", "error", err)
	}
	return nil
}

func (access *UAccess) deleteUnconfirmedMeta(batch *leveldb.Batch, block *ledger.AccountBlock) (err error) {
	if err = access.store.DeleteMeta(batch, &block.ToAddress, &block.Hash); err != nil {
		access.log.Error("DeleteMeta", "error", err)
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
		block, err := access.chain.GetAccountBlockByHash(v)
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
		AccountAddress: addr,
		TotalNumber:    number,
		TokenInfoMap:   infoMap,
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
		block, err := access.chain.GetAccountBlockByHash(v)
		if err != nil || block == nil {
			access.log.Error("ContractWorker.GetBlockByHash", "error", err)
			continue
		}
		result[i] = block
	}

	return result, nil
}

func (access *UAccess) GetCommonAccTokenInfoMap(addr *types.Address) (map[types.TokenTypeId]*TokenInfo, uint64, error) {
	infoMap := make(map[types.TokenTypeId]*TokenInfo)
	hashList, err := access.store.GetHashList(addr)
	if err != nil {
		access.log.Error("GetCommonAccTokenInfoMap.GetHashList", "error", err)
		return nil, 0, err
	}
	for _, v := range hashList {
		block, err := access.chain.GetAccountBlockByHash(v)
		if err != nil || block == nil {
			access.log.Error("ContractWorker.GetBlockByHash", "error", err)
			continue
		}
		ti, ok := infoMap[block.TokenId]
		if !ok {
			token, err := access.chain.GetTokenInfoById(&block.TokenId)
			if err != nil {
				access.log.Error("func GetUnconfirmedAccount.GetByTokenId failed", "error", err)
				return nil, 0, err
			}
			infoMap[block.TokenId].Token = token.Mintage
			infoMap[block.TokenId].TotalAmount = *block.Amount
		} else {
			ti.TotalAmount.Add(&ti.TotalAmount, block.Amount)
		}

	}
	return infoMap, uint64(len(hashList)), err
}
