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

const (
	MaxReceiveErrCount = 3
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

func (access *UAccess) GetContractAddrListByGid(gid *types.Gid) (addrList []types.Address, err error) {
	return access.store.GetContractAddrList(gid)
}

func (access *UAccess) WriteContractAddrToGid(batch *leveldb.Batch, gid types.Gid, address types.Address) error {
	var addrList []types.Address
	var err error

	addrList, err = access.GetContractAddrListByGid(&gid)
	if addrList == nil && err != nil {
		access.log.Error("GetMeta", "error", err)
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

func (access *UAccess) writeOnroadMeta(batch *leveldb.Batch, block *ledger.AccountBlock) (err error) {
	if block.IsSendBlock() {
		// call from the common WriteOnroad func to add new onRoadTx
		return access.store.WriteMeta(batch, &block.ToAddress, &block.Hash, 0)
	} else {
		// call from the RevertOnroad(revert) func
		addr := &block.AccountAddress
		hash := &block.FromBlockHash

		value, err := access.store.GetMeta(addr, hash)
		if len(value) == 0 {
			if err == nil {
				if count, err := access.store.GetReceiveErrCount(hash, addr); err == nil {
					if block.BlockType == ledger.BlockTypeReceiveError {
						if err := access.store.DecreaseReceiveErrCount(batch, hash, addr); err != nil {
							return err
						}
						return access.store.WriteMeta(batch, addr, hash, count-1)
					} else {
						return access.store.WriteMeta(batch, addr, hash, count)
					}
				}
				return err
			}
			return errors.New("GetMeta error:" + err.Error())
		} else {
			// count := uint8(value[0])
			if count, err := access.store.GetReceiveErrCount(hash, addr); err == nil {
				if err := access.store.DecreaseReceiveErrCount(batch, hash, addr); err != nil {
					return err
				}
				return access.store.WriteMeta(batch, addr, hash, count-1)
			}
			return err
		}
	}
}

func (access *UAccess) deleteOnroadMeta(batch *leveldb.Batch, block *ledger.AccountBlock) (err error) {
	if block.IsReceiveBlock() {
		// call from the WriteOnroad func to handle the onRoadTx's receiveBlock
		addr := &block.AccountAddress
		hash := &block.FromBlockHash

		code, err := access.Chain.AccountType(&block.AccountAddress)
		switch code {
		case ledger.AccountTypeGeneral, ledger.AccountTypeNotExist:
			return access.store.DeleteMeta(batch, addr, hash)
		case ledger.AccountTypeContract:
			value, err := access.store.GetMeta(addr, hash)
			if len(value) == 0 {
				if err == nil {
					access.log.Info("the corresponding sendBlock has already been delete")
					return nil
				}
				return errors.New("GetMeta error:" + err.Error())
			}
			//count := uint8(value[0])
			if count, err := access.store.GetReceiveErrCount(hash, addr); err == nil {
				if block.BlockType == ledger.BlockTypeReceiveError {
					if count < MaxReceiveErrCount {
						if err := access.store.IncreaseReceiveErrCount(batch, hash, addr); err != nil {
							return err
						}
						return access.store.WriteMeta(batch, addr, hash, count+1)
					}
				}
				return access.store.DeleteMeta(batch, addr, hash)
			} else {
				return err
			}
		default:
			if err != nil {
				access.log.Error("AccountType", "error", err)
			}
			return errors.New("AccountType error ")
		}
	} else {
		// call from the  RevertOnroad(revert) func to handle sendBlock
		if err := access.store.DeleteReceiveErrCount(batch, &block.Hash, &block.ToAddress); err != nil {
			return err
		}
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
