package unconfirmed

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/unconfirmed/worker"
	"github.com/vitelabs/go-vite/vitedb"
	"math/big"
)

type UnconfirmedAccess struct {
	store           *vitedb.Unconfirmed
	commonTxWorkers *map[types.Address]*worker.AutoReceiveWorker
	contractWorkers *map[types.Address]*worker.ContractWorker
	log             log15.Logger
}

func NewUnconfirmedAccess(commonTxWorkers *map[types.Address]*worker.AutoReceiveWorker, contractWorkers *map[types.Address]*worker.ContractWorker) *UnconfirmedAccess {
	return &UnconfirmedAccess{
		store:           vitedb.NewUnconfirmedDB(),
		commonTxWorkers: commonTxWorkers,
		contractWorkers: contractWorkers,
		log:             slog.New("w", "unconfirmedAccess"),
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
	default:
		break
	}
}

func (access *UnconfirmedAccess) WriteUnconfirmed(batch *leveldb.Batch, writeType bool, block *AccountBlock) error {
	if writeType == true {
		// writeType == true: add new UnconfirmedMeta
		access.NewSignalToWorker(block)
		if err := access.WriteUnconfirmedMeta(batch, block); err != nil {
			access.log.Error("WriteUnconfirmedMeta", "error", err)
			return err
		}

	} else {
		// writeType == false: delete processed UnconfirmedMeta

	}
	return nil
}

func (access *UnconfirmedAccess) WriteUnconfirmedMeta(batch *leveldb.Batch, block *AccountBlock) error {
	return nil
}

func (access *UnconfirmedAccess) DeleteUnconfirmedMeta(batch *leveldb.Batch, block *AccountBlock) error {
	return nil
}

func (access *UnconfirmedAccess) GetUnconfirmedHashs(index, num, count int, addr *types.Address) ([]*types.Hash, error) {
	meta, err := access.store.GetUnconfirmedMeta(addr)
	if err != nil {
		return nil, err
	}
	numberInt := big.NewInt(int64((index + num) * count))
	if big.NewInt(0).Cmp(meta.TotalNumber) == 0 {
		return nil, nil
	}
	hashList, err := access.store.GetAccTotalHashList(addr)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}
	if numberInt.Cmp(meta.TotalNumber) == 1 {
		return hashList[index*count:], nil
	}
	return hashList[index*count : (index+num)*count], nil

func (access *UnconfirmedAccess) GetAddressListByGid(gid string) (addressList []*types.Address, err error) {
	return nil, nil
}

func (access *UnconfirmedAccess) GetUnconfirmedBlockList([]*types.Hash) ([]*AccountBlock, error) {
	return nil, nil
}
