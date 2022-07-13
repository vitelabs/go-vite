package onroad_pool

import (
	"github.com/vitelabs/go-vite/v2/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
)

type orHeightValue struct {
	Height uint64
	Hashes []*orHeightSubValue
}
type orHeightSubValue struct {
	Hash     types.Hash
	SubIndex *uint8
}

func newOrHeightValue(hh orHashHeight) orHeightValue {
	return orHeightValue{
		Height: hh.Height,
		Hashes: []*orHeightSubValue{
			{
				Hash:     hh.Hash,
				SubIndex: hh.SubIndex,
			}},
	}
}

func (hv *orHeightValue) push(hh orHashHeight) error {
	if hh.Height != hv.Height {
		return errors.New("orHeightValue push height not match")
	}
	for _, sub := range hv.Hashes {
		if sub.Hash == hh.Hash {
			return errors.New("orHeight push hash duplicated")
		}
	}
	hv.Hashes = append(hv.Hashes, &orHeightSubValue{Hash: hh.Hash, SubIndex: hh.SubIndex})
	return nil
}
func (hv *orHeightValue) remove(hh orHashHeight) error {
	if hh.Height != hv.Height {
		return errors.New("orHeightValue push height not match")
	}

	j := -1
	for i, sub := range hv.Hashes {
		if sub.Hash == hh.Hash {
			j = i
			break
		}
	}
	if j >= 0 {
		hv.Hashes[j] = hv.Hashes[len(hv.Hashes)-1]
		hv.Hashes = hv.Hashes[:len(hv.Hashes)-1]
		return nil
	}
	return errors.New("[remove]the item not found")
}
func (hv orHeightValue) isEmpty() bool {
	return len(hv.Hashes) == 0
}
func (hv *orHeightValue) dirtySubIndex() []*orHeightSubValue {
	var result []*orHeightSubValue
	for _, sub := range hv.Hashes {
		if sub.SubIndex == nil {
			result = append(result, sub)
		}
	}
	return result
}
func (hv orHeightValue) minIndex() (*orHashHeight, error) {
	if len(hv.Hashes) == 0 {
		return nil, errors.New("height value is empty")
	}
	var min *orHeightSubValue

	for _, sub := range hv.Hashes {
		if sub.SubIndex == nil {
			return nil, errors.New("sub index is nil")
		}
		if min == nil {
			min = sub
		} else {
			if *sub.SubIndex < *min.SubIndex {
				min = sub
			}
		}
	}

	return &orHashHeight{
		Hash:     min.Hash,
		Height:   hv.Height,
		SubIndex: min.SubIndex,
	}, nil

}

type orHashHeight struct {
	Hash types.Hash

	Height   uint64
	SubIndex *uint8

	cachedBlock *ledger.AccountBlock
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
		index := uint8(0)
		or.hashHeight = orHashHeight{
			Hash:     block.Hash,
			Height:   block.Height,
			SubIndex: &index,
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
			return nil, ErrFindCompleteBlock
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
