package unconfirmed

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/unconfirmed/worker"
	"github.com/vitelabs/go-vite/vitedb"
)

type UnconfirmedAccess struct {
	store           *vitedb.Unconfirmed
	commonTxWorkers *map[types.Address]*worker.AutoReceiveWorker
	contractWorkers *map[types.Address]*worker.ContractWorker
}

func NewUnconfirmedAccess(commonTxWorkers *map[types.Address]*worker.AutoReceiveWorker, contractWorkers *map[types.Address]*worker.ContractWorker) *UnconfirmedAccess {
	return &UnconfirmedAccess{
		store:           vitedb.NewUnconfirmed(),
		commonTxWorkers: commonTxWorkers,
		contractWorkers: contractWorkers,
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

// writeType == true: add new UnconfirmedMeta
// writeType == false: delete processed UnconfirmedMeta
func (access *UnconfirmedAccess) WriteUnconfirmed(writeType bool, block *AccountBlock) error {
	if writeType == true {
		access.NewSignalToWorker(block)
	} else {

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
	return nil, nil
}

func (access *UnconfirmedAccess) GetAddressListByGid(gid string) (addressList []*types.Address, err error) {
	return nil, nil
}

func (access *UnconfirmedAccess) GetUnconfirmedBlockList([]*types.Hash) ([]*AccountBlock, error) {
	return nil, nil
}
