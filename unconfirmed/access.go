package unconfirmed

import (
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/unconfirmed/worker"
	"github.com/vitelabs/go-vite/vitedb"
)

type UnconfirmedAccess struct {
	store                *vitedb.UnconfirmedDB
	commonTxWorkers      *map[types.Address]*worker.CommonTxWorker
	contractWorkers      *map[types.Address]*worker.ContractWorker
	commonAccountInfoMap *map[types.Address]*CommonAccountInfo

	log log15.Logger
}

func NewUnconfirmedAccess(commonTxWorkers *map[types.Address]*worker.CommonTxWorker,
	contractWorkers *map[types.Address]*worker.ContractWorker,
	commonAccountInfo *map[types.Address]*CommonAccountInfo) *UnconfirmedAccess {
	return &UnconfirmedAccess{
		store:                vitedb.NewUnconfirmedDB(),
		commonTxWorkers:      commonTxWorkers,
		contractWorkers:      contractWorkers,
		commonAccountInfoMap: commonAccountInfo,
		log:                  slog.New("w", "unconfirmedAccess"),
	}
}

func (access *UnconfirmedAccess) NewSignalToWorker(block *AccountBlock) {
	select {
	case w, ok := (*access.contractWorkers)[*block.To]:
		if ok && w.Status() != worker.Stop {
			(*access.contractWorkers)[*block.To].NewUnconfirmedTxAlarm()
		}
	case w, ok := (*access.commonTxWorkers)[*block.To]:
		if ok && w.Status() != worker.Stop {
			(*access.commonTxWorkers)[*block.To].NewUnconfirmedTxAlarm()
		}
	}
}

func (access *UnconfirmedAccess) GetAddrListByGid(gid []byte) (addrList []*types.Address, err error) {
	return nil, nil
}

func (access *UnconfirmedAccess) WriteUnconfirmed(writeType bool, batch *leveldb.Batch, block *AccountBlock) error {
	select {
	case writeType == true:
		// writeType == true: add new UnconfirmedMeta
		if err := access.WriteUnconfirmedMeta(batch, block); err != nil {
			access.log.Error("WriteUnconfirmedMeta", "error", err)
			return err
		}

		// fixme: whether need to wait the block insert into chain and try the following
		access.NewSignalToWorker(block)

	case writeType == false:
		// writeType == false: delete processed UnconfirmedMeta
		if err := access.DeleteUnconfirmedMeta(batch, block); err != nil {
			access.log.Error("DeleteUnconfirmedMeta", "error", err)
			return err
		}
	}

	// fixme: whether need to wait the block insert into chain and try the following
	if err := access.UpdateCommonAccInfo(writeType, block); err != nil {
		access.log.Error("UpdateCommonAccInfo", "error", err)
	}

	return nil
}

func (access *UnconfirmedAccess) WriteUnconfirmedMeta(batch *leveldb.Batch, block *AccountBlock) (err error) {
	if err = access.store.WriteMeta(batch, block.To, block.Hash); err != nil {
		access.log.Error("WriteMeta", "error", err)
	}
	return nil
}

func (access *UnconfirmedAccess) DeleteUnconfirmedMeta(batch *leveldb.Batch, block *AccountBlock) (err error) {
	if err = access.store.DeleteMeta(batch, block.To, block.Hash); err != nil {
		access.log.Error("DeleteMeta", "error", err)
	}
	return nil
}

func (access *UnconfirmedAccess) GetUnconfirmedHashs(index, num, count uint64, addr *types.Address) ([]*types.Hash, error) {
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

func (access *UnconfirmedAccess) GetUnconfirmedBlocks(index, num, count uint64, addr *types.Address) (blockList []*AccountBlock, err error) {
	hashList, err := access.GetUnconfirmedHashs(index, num, count, addr)
	if err != nil {
		return nil, err
	}
	for _, v := range hashList {
		block, err := GetAccountChainAccess.GetBlockByHash(v)
		if err != nil || block == nil {
			access.log.Error("ContractWorker.GetBlockByHash", "error", err)
			continue
		}
		blockList = append(blockList, block)
	}
	return nil, nil
}

func (access *UnconfirmedAccess) GetCommonAccInfo(addr *types.Address) (info *CommonAccountInfo, err error) {
	if _, ok := (*access.commonAccountInfoMap)[*addr]; !ok {
		if err = access.LoadCommonAccInfo(addr); err != nil {
			access.log.Error("GetCommonAccInfo.LoadCommonAccInfo", "error", err)
			return nil, err
		}
	}
	info, _ = (*access.commonAccountInfoMap)[*addr]
	return info, nil
}

func (access *UnconfirmedAccess) LoadCommonAccInfo(addr *types.Address) error {
	if _, ok := (*access.commonAccountInfoMap)[*addr]; !ok {
		number, err := access.store.GetCountByAddress(addr)
		if err != nil {
			return err
		}
		infoMap, err := access.GetCommonAccTokenInfoMap(addr)
		if err != nil {
			return err
		}
		info := &CommonAccountInfo{
			AccountAddress: addr,
			TotalNumber:    number,
			TokenInfoMap:   *infoMap,
		}
		(*access.commonAccountInfoMap)[*addr] = info
	}
	return nil
}

func (access *UnconfirmedAccess) GetCommonAccTokenInfoMap(addr *types.Address) (*map[*types.TokenTypeId]*TokenInfo, error) {
	infoMap := make(map[*types.TokenTypeId]*TokenInfo)
	hashList, err := access.store.GetHashList(addr)
	if err != nil {
		access.log.Error("GetCommonAccTokenInfoMap.GetHashList", "error", err)
		return nil, err
	}
	for _, v := range hashList {
		block, err := GetAccountChainAccess.GetBlockByHash(v)
		if err != nil || block == nil {
			access.log.Error("ContractWorker.GetBlockByHash", "error", err)
			continue
		}
		if _, ok := infoMap[block.TokenId]; !ok {
			token, err := GetTokenAccess.GetByTokenId(block.TokenId)
			if err != nil {
				access.log.Error("func GetUnconfirmedAccount.GetByTokenId failed", "error", err)
				return nil, err
			}
			infoMap[block.TokenId].Token = token.Mintage
			infoMap[block.TokenId].TotalAmount = block.Amount
		}
		infoMap[block.TokenId].TotalAmount.Add(block.Amount)
	}
	return &infoMap, err
}

func (access *UnconfirmedAccess) UpdateCommonAccInfo(writeType bool, block *AccountBlock) error {
	tiMap, ok := (*access.commonAccountInfoMap)[*block.To]
	if !ok {
		access.log.Info("UpdateCommonAccInfo：no memory maintenance:",
			"reason", "send-to address doesn't belong to current manager")
		return nil
	}
	select {
	case writeType == true:
		ti, ok := tiMap.TokenInfoMap[block.TokenId]
		if !ok {
			token, err := GetTokenAccess.GetByTokenId(block.TokenId)
			if err != nil {
				return errors.New("func UpdateCommonAccInfo.GetByTokenId failed" + err)
			}
			tiMap.TokenInfoMap[block.TokenId].Token = token.Mintage
			tiMap.TokenInfoMap[block.TokenId].TotalAmount = block.Amount
		} else {
			ti.TotalAmount.Add(ti.TotalAmount, block.Amount)
		}

	case writeType == false:
		ti, ok := tiMap.TokenInfoMap[block.TokenId]
		if !ok {
			return errors.New("find no memory tokenInfo, so can't update when writeType is false")
		} else {
			if ti.TotalAmount.Cmp(block.Amount) == -1 {
				return errors.New("conflict with the memory info, so can't update when writeType is false")
			}
			if ti.TotalAmount.Cmp(block.Amount) == 0 {
				delete(tiMap.TokenInfoMap, block.TokenId)
			} else {
				ti.TotalAmount.Sub(ti.TotalAmount, block.Amount)
			}
		}
	}
	return nil
}
