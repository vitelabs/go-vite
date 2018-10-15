package model

import (
	"container/list"
	"math/big"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vitepb"
)

type OnroadMeta struct {
	Gid     []byte
	Address types.Address
	Hash    types.Hash
}

type OnroadAccountInfo struct {
	mutex               sync.RWMutex
	AccountAddress      *types.Address
	TotalNumber         uint64
	TokenBalanceInfoMap map[types.TokenTypeId]*TokenBalanceInfo
}

type TokenBalanceInfo struct {
	TotalAmount big.Int
	Number      uint64
}

type onroadBlocksCache struct {
	blocks     *list.List
	currentEle *list.Element
	listMutex  sync.RWMutex

	referenceCount int
	referenceMutex sync.Mutex
}

func (o onroadBlocksCache) getReferenceCount() int {
	o.referenceMutex.Lock()
	defer o.referenceMutex.Unlock()
	return o.referenceCount
}

func (c *onroadBlocksCache) toOnroadAccountInfo(addr types.Address) *OnroadAccountInfo {

	c.listMutex.RLock()
	defer c.listMutex.RUnlock()
	ele := c.blocks.Front()
	var ca OnroadAccountInfo
	ca.AccountAddress = &addr
	infoMap := make(map[types.TokenTypeId]*TokenBalanceInfo)
	for ele != nil {

		block := ele.Value.(*ledger.AccountBlock)
		ti, ok := infoMap[block.TokenId]
		if !ok {
			var tinfo TokenBalanceInfo
			tinfo.TotalAmount = *block.Amount
			infoMap[block.TokenId] = &tinfo
		} else {
			ti.TotalAmount.Add(&ti.TotalAmount, block.Amount)
		}

		infoMap[block.TokenId].Number += 1

		ele = ele.Next()
	}

	ca.TotalNumber = uint64(c.blocks.Len())
	ca.TokenBalanceInfoMap = infoMap
	return &ca
}

func (c *onroadBlocksCache) ResetCursor() {
	c.listMutex.RLock()
	defer c.listMutex.RUnlock()
	c.currentEle = c.blocks.Front()
}

func (c *onroadBlocksCache) GetNextTx() *ledger.AccountBlock {
	if c.currentEle == nil {
		return nil
	}
	c.listMutex.Lock()
	defer c.listMutex.Unlock()
	block := c.currentEle.Value.(*ledger.AccountBlock)
	c.currentEle = c.currentEle.Next()
	return block
}

func (c *onroadBlocksCache) addTx(b *ledger.AccountBlock) {
	c.listMutex.Lock()
	defer c.listMutex.Unlock()
	c.blocks.PushBack(b)
	if c.currentEle == nil {
		c.currentEle = c.blocks.Front()
	}
}

func (c *onroadBlocksCache) rmTx(b *ledger.AccountBlock) {
	if b == nil {
		return
	}
	c.listMutex.Lock()
	defer c.listMutex.Unlock()
	ele := c.blocks.Front()
	for ele != nil {
		next := ele.Next()
		if ele.Value.(*ledger.AccountBlock).Hash == b.FromBlockHash {
			c.blocks.Remove(ele)
			if ele == c.currentEle {
				c.currentEle = next
			}
		}
		ele = next
	}
}

func (c *onroadBlocksCache) addReferenceCount() int {
	c.referenceMutex.Lock()
	defer c.referenceMutex.Unlock()
	c.referenceCount += 1
	return c.referenceCount
}

func (c *onroadBlocksCache) subReferenceCount() int {
	c.referenceMutex.Lock()
	defer c.referenceMutex.Unlock()
	c.referenceCount -= 1
	return c.referenceCount
}

func AddrListDbSerialize(addrList []types.Address) ([]byte, error) {
	var aList [][]byte
	for _, addr := range addrList {
		aList = append(aList, addr.Bytes())
	}
	var addrListPB = &vitepb.AddressList{
		AddressList: aList,
	}
	data, err := proto.Marshal(addrListPB)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func AddrListDbDeserialize(buf []byte) ([]types.Address, error) {
	var addrListPB = &vitepb.AddressList{}
	if err := proto.Unmarshal(buf, addrListPB); err != nil {
		return nil, err
	}
	addrList := make([]types.Address, len(addrListPB.AddressList))
	for k, addrPB := range addrListPB.AddressList {
		addr, err := types.BytesToAddress(addrPB)
		if err != nil {
			return nil, err
		}
		addrList[k] = addr
	}
	return addrList, nil
}
