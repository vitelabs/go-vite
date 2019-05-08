package onroad_pool

import (
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type orHashHeight struct {
	Hash types.Hash

	Height   uint64
	SubIndex *uint8
}

type onRoadList []*orHashHeight

func (oList onRoadList) Len() int { return len(oList) }
func (oList onRoadList) Swap(i, j int) {
	oList[i], oList[j] = oList[j], oList[i]
}
func (oList onRoadList) Less(i, j int) bool {
	if oList[i].Height < oList[j].Height {
		return true
	}
	if oList[i].Height == oList[j].Height && oList[i].SubIndex != nil && oList[j].SubIndex != nil {
		if *oList[i].SubIndex < *oList[j].SubIndex {
			return true
		}
	}
	return false
}

type OnRoadBlock struct {
	caller     types.Address
	orAddr     types.Address
	hashHeight orHashHeight

	block *ledger.AccountBlock
}

func LedgerBlockToOnRoad(chain chainReader, block *ledger.AccountBlock) (*OnRoadBlock, error) {
	or := &OnRoadBlock{
		block: block,
	}

	if block.IsSendBlock() {
		or.caller = block.AccountAddress
		or.orAddr = block.ToAddress
		or.hashHeight = orHashHeight{
			Hash:   block.Hash,
			Height: block.Height,
		}
	} else {
		fromBlock, err := chain.GetAccountBlockByHash(block.FromBlockHash)
		if err != nil {
			return nil, err
		}
		if fromBlock == nil {
			return nil, errors.New("failed to find send")
		}
		or.caller = fromBlock.AccountAddress
		or.orAddr = fromBlock.ToAddress
		or.hashHeight = orHashHeight{
			Hash:   fromBlock.Hash,
			Height: fromBlock.Height,
		}
	}

	if types.IsContractAddr(or.caller) {
		completeBlock, err := chain.GetCompleteBlockByHash(or.hashHeight.Hash)
		if err != nil {
			return nil, err
		}
		if completeBlock == nil {
			return nil, errors.New("failed to find complete send's parent receive")
		}
		or.hashHeight.Height = completeBlock.Height // refer to its parent receive's height

		for k, v := range completeBlock.SendBlockList {
			if v.Hash == or.hashHeight.Hash {
				idx := uint8(k)
				or.hashHeight.SubIndex = &idx
				break
			}
		}
	}

	return or, nil
}

type PendingOnRoadList []*OnRoadBlock

func (pList PendingOnRoadList) Len() int { return len(pList) }
func (pList PendingOnRoadList) Swap(i, j int) {
	pList[i], pList[j] = pList[j], pList[i]
}
func (pList PendingOnRoadList) Less(i, j int) bool {
	if pList[i].hashHeight.Height < pList[j].hashHeight.Height {
		return true
	}
	if pList[i].hashHeight.Height == pList[j].hashHeight.Height && pList[i].hashHeight.SubIndex != nil && pList[j].hashHeight.SubIndex != nil {
		if *pList[i].hashHeight.SubIndex < *pList[j].hashHeight.SubIndex {
			return true
		}
	}
	return false
}
