package model

import (
	"bytes"
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

func (access *UAccess) GetContractAddrListWithoutPrecompiledByGid(gid *types.Gid) ([]types.Address, error) {
	addrList, err := access.store.GetContractAddrList(gid)
	if err != nil {
		access.log.Error("GetContractAddrListWithoutPrecompiledByGid", "error", err)
		return nil, err
	}

	//if *gid == types.DELEGATE_GID {
	//	addrList = append(addrList, types.PrecompiledContractAddressList...)
	//}
	return addrList, nil
}

func (access *UAccess) WriteContractAddrToGid(batch *leveldb.Batch, gid types.Gid, address types.Address) error {
	var addrList []types.Address
	var err error

	addrList, err = access.store.GetContractAddrList(&gid)
	if err != nil {
		access.log.Error("WriteContractAddrToGid", "error", err)
		return err
	} else {
		for _, v := range addrList {
			if v == address {
				return nil
			}
		}
		addrList = append(addrList, address)
		return access.store.WriteGidAddrList(batch, &gid, addrList)
	}
}

func (access *UAccess) DeleteContractAddrFromGid(batch *leveldb.Batch, gid types.Gid, address types.Address) error {
	var addrList []types.Address
	var err error

	addrList, err = access.store.GetContractAddrList(&gid)
	if addrList == nil || err != nil {
		access.log.Error("DeleteContractAddrFromGid", "error", err)
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
			return errors.New("GetReceiveErrCount error" + recvErr.Error())
		}
		if block.BlockType == ledger.BlockTypeReceiveError && len(recvErrList) <= 0 {
			return errors.New("conflict, revert receiveError but recvErrCount is 0")
		}

		if block.BlockType == ledger.BlockTypeReceive {
			if access.Chain.IsSuccessReceived(addr, hash) {
				access.log.Info("the corresponding sendBlock has already been delete")
				return access.store.WriteMeta(batch, addr, hash)
			}
			return errors.New("conflict, revert receiveBlock but referred send still in onroad")
		}
		return nil
	}
}

func (access *UAccess) deleteOnroadMeta(batch *leveldb.Batch, block *ledger.AccountBlock) error {
	if block.IsReceiveBlock() {
		// call from the WriteOnroad func to handle the onRoadTx's receiveBlock
		addr := &block.AccountAddress
		hash := &block.FromBlockHash

		if access.Chain.IsSuccessReceived(&block.AccountAddress, &block.FromBlockHash) {
			access.log.Info("the corresponding sendBlock has already been delete")
			return nil
		}
		if block.BlockType == ledger.BlockTypeReceive {
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
		access.log.Error("GetOnroadHashs", "error", err)
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

func (access *UAccess) GetOnroadBlocks(index, num, count uint64, addr *types.Address) (blockList []*ledger.AccountBlock, err error) {
	hashList, err := access.GetOnroadHashs(index, num, count, addr)
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

func (access *UAccess) GetAllOnroadBlocks(addr types.Address) (blockList []*ledger.AccountBlock, err error) {
	hashList, err := access.store.GetHashList(&addr)
	if err != nil {
		return nil, err
	}
	result := make([]*ledger.AccountBlock, len(hashList))

	count := 0
	for i, v := range hashList {
		block, err := access.Chain.GetAccountBlockByHash(v)
		if err != nil || block == nil {
			access.log.Error("ContractWorker.GetBlockByHash", "error", err)
			continue
		}
		result[i] = block
		count++
	}

	return result[0:count], nil
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
