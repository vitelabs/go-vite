package onroad_pool

import (
	"fmt"

	"github.com/vitelabs/go-vite/v2/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/log15"
)

var onroadPoolLog = log15.New("onroadPool", nil)

type orHeightValue []*OnroadTx

func newOrHeightValueFromOnroadTxs(txs []OnroadTx) (orHeightValue, error) {
	result := make([]*OnroadTx, len(txs))
	for index := range txs {
		result[index] = &txs[index]
	}
	return result, nil
}

func (hv orHeightValue) isEmpty() bool {
	return len(hv) == 0
}
func (hv orHeightValue) dirtyTxs() []*OnroadTx {
	var result []*OnroadTx
	for _, sub := range hv {
		if sub.FromIndex == nil {
			result = append(result, sub)
		}
	}
	return result
}
func (hv orHeightValue) minTx() (*OnroadTx, error) {
	if len(hv) == 0 {
		return nil, errors.New("height value is empty")
	}
	var min *OnroadTx

	for _, sub := range hv {
		if sub.FromIndex == nil {
			return nil, errors.New("sub index is nil")
		}
		tmp := sub
		if min == nil {
			min = tmp
		} else {
			if *sub.FromIndex < *min.FromIndex {
				min = tmp
			}
		}
	}

	return min, nil
}

type orHashHeight struct {
	Hash types.Hash

	Height   uint64
	SubIndex *uint32

	// @todo delete
	cachedBlock *ledger.AccountBlock
}

func (or orHashHeight) String() string {
	if or.SubIndex == nil {
		return fmt.Sprintf("orHashHeight: hash=%s,height=%d,subIndex=nil", or.Hash, or.Height)
	} else {
		return fmt.Sprintf("orHashHeight: hash=%s,height=%d,subIndex=%d", or.Hash, or.Height, *or.SubIndex)
	}
}

func newOrHashHeightFromOnroadTx(tx *OnroadTx) *orHashHeight {
	return &orHashHeight{
		Hash:        tx.FromHash,
		Height:      tx.FromHeight,
		SubIndex:    tx.FromIndex,
		cachedBlock: nil,
	}
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

// func ledgerBlockToOnRoad(inserOrRollback bool, block *ledger.AccountBlock, fromBlock *ledger.AccountBlock) (*OnRoadBlock, error) {
// 	or := &OnRoadBlock{
// 		block: block,
// 	}

// 	if block.IsSendBlock() {
// 		or.caller = block.AccountAddress
// 		or.orAddr = block.ToAddress
// 		index := uint8(0)
// 		or.hashHeight = orHashHeight{
// 			Hash:     block.Hash,
// 			Height:   block.Height,
// 			SubIndex: &index,
// 		}
// 	} else {
// 		if fromBlock == nil {
// 			return nil, errors.New("failed to find send")
// 		}
// 		or.caller = fromBlock.AccountAddress
// 		or.orAddr = fromBlock.ToAddress
// 		or.hashHeight = orHashHeight{
// 			Hash:   fromBlock.Hash,
// 			Height: fromBlock.Height,
// 		}
// 	}
// }

func LedgerBlockToOnRoad(chain chainReader, block *ledger.AccountBlock) (*OnRoadBlock, error) {
	or := &OnRoadBlock{
		block: block,
	}

	if block.IsSendBlock() {
		or.caller = block.AccountAddress
		or.orAddr = block.ToAddress
		index := uint32(0)
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
				idx := uint32(k)
				or.hashHeight.SubIndex = &idx
				break
			}
		}
	} else {
		index := uint32(0)
		or.hashHeight.SubIndex = &index
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
