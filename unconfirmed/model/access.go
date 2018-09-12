package model

import (
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"path/filepath"
)

type UAccess struct {
	store                *UnconfirmedSet
	Chain                *chain.Chain
	commonAccountInfoMap map[types.Address]*CommonAccountInfo
	newCommonTxLis       map[types.Address]func()
	newContractLis       map[types.Gid]func()

	log log15.Logger
}

func NewUAccess(chain *chain.Chain, dataDir string) *UAccess {
	uAccess := &UAccess{
		Chain:                chain,
		commonAccountInfoMap: make(map[types.Address]*CommonAccountInfo),
		newCommonTxLis:       make(map[types.Address]func()),
		newContractLis:       make(map[types.Gid]func()),
		log:                  log15.New("w", "access"),
	}
	dbDir := filepath.Join(dataDir, "chain")
	db, err := leveldb.OpenFile(dbDir, nil)
	if err != nil {
		uAccess.log.Error("ChainDb not find or create DB failed")
	}
	uAccess.store = NewUnconfirmedSet(db)
	return uAccess
}

func (access *UAccess) AddCommonTxLis(addr *types.Address, f func()) {
	access.newCommonTxLis[*addr] = f
}
func (access *UAccess) RemoveCommonTxLis(addr *types.Address) {
	delete(access.newCommonTxLis, *addr)
}

func (access *UAccess) AddContractLis(gid *types.Gid, f func()) {
	access.newContractLis[*gid] = f
}
func (access *UAccess) RemoveContractLis(gid *types.Gid) {
	delete(access.newContractLis, *gid)
}

func (access *UAccess) NewSignalToWorker(block *ledger.AccountBlock) {
	if block.IsContractTx() {
		if f, ok := access.newContractLis[*block.Gid]; ok {
			f()
		}
	} else {
		if f, ok := access.newCommonTxLis[block.ToAddress]; ok {
			f()
		}
	}
}

func (access *UAccess) GetAddrListByGid(gid *types.Gid) (addrList []*types.Address, err error) {
	return nil, nil
}

func (access *UAccess) WriteUnconfirmed(writeType bool, batch *leveldb.Batch, block *ledger.AccountBlock) error {
	select {
	case writeType == true:
		// writeType == true: add new UnconfirmedMeta
		if err := access.WriteUnconfirmedMeta(batch, block); err != nil {
			access.log.Error("WriteUnconfirmedMeta", "error", err)
			return err
		}

		// fixme: whether need to wait the block insert into chain and try the following
		access.NewSignalToWorker(block)

		// todo: maintain the gid-contractAddress into VmDB
		// AddConsensusGroup(group ConsensusGroup...)

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

func (access *UAccess) WriteUnconfirmedMeta(batch *leveldb.Batch, block *ledger.AccountBlock) (err error) {
	if err = access.store.WriteMeta(batch, &block.ToAddress, &block.Hash); err != nil {
		access.log.Error("WriteMeta", "error", err)
	}
	return nil
}

func (access *UAccess) DeleteUnconfirmedMeta(batch *leveldb.Batch, block *ledger.AccountBlock) (err error) {
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
	if _, ok := access.commonAccountInfoMap[*addr]; !ok {
		if err = access.LoadCommonAccInfo(addr); err != nil {
			access.log.Error("GetCommonAccInfo.LoadCommonAccInfo", "error", err)
			return nil, err
		}
	}
	info, _ = access.commonAccountInfoMap[*addr]
	return info, nil
}

func (access *UAccess) LoadCommonAccInfo(addr *types.Address) error {
	if _, ok := access.commonAccountInfoMap[*addr]; !ok {
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
		access.commonAccountInfoMap[*addr] = info
	}
	return nil
}

func (access *UAccess) GetCommonAccTokenInfoMap(addr *types.Address) (*map[types.TokenTypeId]*TokenInfo, error) {
	infoMap := make(map[types.TokenTypeId]*TokenInfo)
	hashList, err := access.store.GetHashList(addr)
	if err != nil {
		access.log.Error("GetCommonAccTokenInfoMap.GetHashList", "error", err)
		return nil, err
	}
	for _, v := range hashList {
		block, err := access.Chain.GetAccountBlockByHash(v)
		if err != nil || block == nil {
			access.log.Error("ContractWorker.GetBlockByHash", "error", err)
			continue
		}
		ti, ok := infoMap[block.TokenId]
		if !ok {
			token, err := access.Chain.GetTokenInfoById(&block.TokenId)
			if err != nil {
				access.log.Error("func GetUnconfirmedAccount.GetByTokenId failed", "error", err)
				return nil, err
			}
			infoMap[block.TokenId].Token = token.Mintage
			infoMap[block.TokenId].TotalAmount = *block.Amount
		} else {
			ti.TotalAmount.Add(&ti.TotalAmount, block.Amount)
		}

	}
	return &infoMap, err
}

func (access *UAccess) UpdateCommonAccInfo(writeType bool, block *ledger.AccountBlock) error {
	tiMap, ok := access.commonAccountInfoMap[block.ToAddress]
	if !ok {
		access.log.Info("UpdateCommonAccInfo：no memory maintenance:",
			"reason", "send-to address doesn't belong to current manager")
		return nil
	}
	select {
	case writeType == true:
		ti, ok := tiMap.TokenInfoMap[block.TokenId]
		if !ok {
			token, err := access.Chain.GetTokenInfoById(&block.TokenId)
			if err != nil {
				return errors.New("func UpdateCommonAccInfo.GetByTokenId failed" + err.Error())
			}
			tiMap.TokenInfoMap[block.TokenId].Token = token.Mintage
			tiMap.TokenInfoMap[block.TokenId].TotalAmount = *block.Amount
		} else {
			ti.TotalAmount.Add(&ti.TotalAmount, block.Amount)
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
				ti.TotalAmount.Sub(&ti.TotalAmount, block.Amount)
			}
		}
	}
	return nil
}
