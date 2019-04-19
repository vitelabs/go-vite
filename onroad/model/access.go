package model

import (
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

type UAccess struct {
	Chain chain.Chain
	store *OnroadSet
	log   log15.Logger
}

func NewUAccess() *UAccess {
	uAccess := &UAccess{
		log: log15.New("w", "uAccess"),
	}
	return uAccess
}

func (access *UAccess) Init(chain chain.Chain) {
	access.Chain = chain
	access.store = NewOnroadSet(chain)
}

func (access *UAccess) GetContractAddrListByGid(gid *types.Gid) ([]types.Address, error) {
	addrList, err := access.store.GetContractAddrList(gid)
	if err != nil {
		access.log.Error("GetContractAddrListByGid.GetContractAddrList", "error", err, "gid", gid)
		return nil, err
	}
	if gid != nil && *gid == types.DELEGATE_GID {
		if len(addrList) <= 0 {
			addrList = make([]types.Address, 0)
		}
		addrList = append(addrList, types.PrecompiledContractAddressList...)
	}
	return addrList, nil
}

func (access *UAccess) WriteContractAddrToGid(batch *leveldb.Batch, gid *types.Gid, address *types.Address) error {
	if gid == nil || address == nil {
		return errors.New("WriteContractAddrToGid failed, gid or address can't be nil")
	}
	if *gid == types.DELEGATE_GID && types.IsPrecompiledContractAddress(*address) {
		return nil
	}

	addrList, err := access.store.GetContractAddrList(gid)
	if err != nil {
		access.log.Error("WriteContractAddrToGid failed", "error", err, "gid", gid, "addr", address)
		return err
	}
	if len(addrList) <= 0 {
		addrList = make([]types.Address, 0)
	}
	for _, v := range addrList {
		if v == *address {
			return nil
		}
	}
	addrList = append(addrList, *address)
	return access.store.WriteGidAddrList(batch, gid, addrList)
}

func (access *UAccess) DeleteContractAddrFromGid(batch *leveldb.Batch, gid *types.Gid, address *types.Address) error {
	if gid == nil || address == nil {
		return errors.New("WriteContractAddrToGid failed, gid or address can't be nil")
	}

	var addrList []types.Address
	var err error

	addrList, err = access.store.GetContractAddrList(gid)
	if err != nil || len(addrList) <= 0 {
		access.log.Error("DeleteContractAddrFromGid.GetContractAddrList failed", "error", err, "gid", gid, "addr", address)
		return err
	}
	for k, v := range addrList {
		if v == *address {
			if k >= len(addrList)-1 {
				addrList = addrList[0:k]
			} else {
				addrList = append(addrList[0:k], addrList[k+1:]...)
			}
			break
		}
	}
	return access.store.WriteGidAddrList(batch, gid, addrList)
}

func (access *UAccess) writeOnroadMeta(batch *leveldb.Batch, block *ledger.AccountBlock) error {
	if block.IsSendBlock() {
		// call from the common WriteOnroad func to add new onRoadTx, sendBlock
		return access.store.WriteMeta(batch, &block.ToAddress, &block.Hash)
	} else {
		// call from the RevertOnroad(revert) func, receiveBlock
		addr := &block.AccountAddress
		hash := &block.FromBlockHash

		recvErrList, recvErr := access.Chain.GetReceiveBlockHeights(hash)
		if recvErr != nil {
			access.log.Error("writeOnroadMeta.RevertOnroad.GetReceiveBlockHeights failed", "err", recvErr, "recvAddr", addr, "fromHash", hash)
			return recvErr
		}
		if block.BlockType == ledger.BlockTypeReceiveError && len(recvErrList) <= 0 {
			recvErr = errors.New("conflict, revert receiveError but recvErrCount is 0")
			access.log.Error(recvErr.Error(), "recvAddr", addr, "fromHash", hash)
			return recvErr
		}

		if block.BlockType == ledger.BlockTypeReceive {
			if access.Chain.IsSuccessReceived(addr, hash) {
				return access.store.WriteMeta(batch, addr, hash)
			}
			recvErr = errors.New("conflict, revert receiveBlock but referred send still in onroad")
			access.log.Error(recvErr.Error(), "recvAddr", addr, "fromHash", hash)
			return recvErr
		}
		return nil
	}
}

func (access *UAccess) deleteOnroadMeta(batch *leveldb.Batch, block *ledger.AccountBlock) error {
	if block.IsReceiveBlock() {
		// call from the WriteOnroad func to handle the onRoadTx's receiveBlock
		addr := &block.AccountAddress
		hash := &block.FromBlockHash
		if block.BlockType == ledger.BlockTypeReceiveError {
			return nil
		}
		if block.BlockType == ledger.BlockTypeReceive {
			if access.Chain.IsSuccessReceived(addr, hash) {
				recvErr := errors.New("conflict, onroad is already successfully received")
				access.log.Error(recvErr.Error(), "recvAddr", addr, "fromHash", hash)
				return recvErr
			}
			return access.store.DeleteMeta(batch, addr, hash)
		}
		return nil
	} else {
		// call from the  RevertOnroad(revert) func to handle sendBlock
		return access.store.DeleteMeta(batch, &block.ToAddress, &block.Hash)
	}
}

func (access *UAccess) GetOnroadHashs(index, num, count uint64, addr *types.Address) ([]*types.Hash, error) {
	totalCount := (index + num) * count
	maxCount, err := access.store.GetCountByAddress(addr)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, errors.New("GetOnroadHashs.GetCountByAddress failed, err:" + err.Error())
	}
	if totalCount > maxCount {
		totalCount = maxCount
	}
	return access.store.GetHashsByCount(totalCount, addr)
}

func (access *UAccess) GetOnroadBlocks(index, num, count uint64, addr *types.Address) (blockList []*ledger.AccountBlock, err error) {
	hashList, err := access.GetOnroadHashs(index, num, count, addr)
	if err != nil {
		return nil, err
	}
	for _, v := range hashList {
		block, err := access.Chain.GetAccountBlockByHash(v)
		if err != nil || block == nil {
			access.log.Info("GetOnroadBlocks.GetBlockByHash", "error", err, "addr", addr, "hash", v)
			continue
		}
		blockList = append(blockList, block)
	}
	return blockList, nil
}

func (access *UAccess) GetCommonAccInfo(addr *types.Address) (info *OnroadAccountInfo, err error) {
	infoMap, number, err := access.GetCommonAccTokenInfoMap(addr)
	if err != nil {
		return nil, err
	}
	info = &OnroadAccountInfo{
		AccountAddress:      addr,
		TotalNumber:         number,
		TokenBalanceInfoMap: infoMap,
	}

	return info, nil
}

func (access *UAccess) GetAllOnroadBlocks(addr *types.Address) (blockList []*ledger.AccountBlock, err error) {
	hashList, err := access.store.GetHashList(addr)
	if err != nil {
		return nil, err
	}
	if len(hashList) <= 0 {
		return nil, nil
	}

	result := make([]*ledger.AccountBlock, len(hashList))
	count := 0
	for _, v := range hashList {
		block, err := access.Chain.GetAccountBlockByHash(v)
		if err != nil || block == nil {
			access.log.Info("GetAllOnroadBlocks.GetBlockByHash", "error", err, "addr", addr, "hash", v)
			continue
		}
		result[count] = block
		count++
	}

	return result[0:count], nil
}

func (access *UAccess) GetCommonAccTokenInfoMap(addr *types.Address) (map[types.TokenTypeId]*TokenBalanceInfo, uint64, error) {
	hashList, err := access.store.GetHashList(addr)
	if err != nil {
		access.log.Error("GetCommonAccTokenInfoMap.GetHashList", "error", err, "addr", addr)
		return nil, 0, err
	}
	if len(hashList) <= 0 {
		return nil, 0, nil
	}

	infoMap := make(map[types.TokenTypeId]*TokenBalanceInfo)
	for _, v := range hashList {
		block, err := access.Chain.GetAccountBlockByHash(v)
		if err != nil || block == nil {
			access.log.Info("GetCommonAccTokenInfoMap.GetBlockByHash", "error", err, "addr", addr, "hash", v)
			continue
		}
		ti, ok := infoMap[block.TokenId]
		if !ok {
			var tinfo TokenBalanceInfo
			tinfo.TotalAmount = *block.Amount
			infoMap[block.TokenId] = &tinfo
		} else {
			ti.TotalAmount.Add(&ti.TotalAmount, block.Amount)
		}
		infoMap[block.TokenId].Number += 1

	}
	return infoMap, uint64(len(hashList)), nil
}
