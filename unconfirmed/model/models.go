package model

import (
	"container/list"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/contracts"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vitepb"
	"math/big"
	"sync"
)

type UnconfirmedMeta struct {
	Gid     []byte
	Address types.Address
	Hash    types.Hash
}

type CommonAccountInfo struct {
	AccountAddress      *types.Address
	TotalNumber         uint64
	TokenBalanceInfoMap map[types.TokenTypeId]*TokenBalanceInfo
}

type TokenBalanceInfo struct {
	Token       contracts.TokenInfo
	TotalAmount big.Int
	Number      uint64
}

type unconfirmedBlocksCache struct {
	blocks     list.List
	currentEle *list.Element

	referenceCount int
	referenceMutex sync.Mutex
}

func (c *unconfirmedBlocksCache) addReferenceCount() int {
	c.referenceMutex.Lock()
	defer c.referenceMutex.Unlock()
	c.referenceCount += 1
	return c.referenceCount
}

func (c *unconfirmedBlocksCache) subReferenceCount() int {
	c.referenceMutex.Lock()
	defer c.referenceMutex.Unlock()
	c.referenceCount -= 1
	return c.referenceCount
}

func (c *unconfirmedBlocksCache) toCommonAccountInfo(GetTokenInfoById func(tti *types.TokenTypeId) (*contracts.TokenInfo, error)) *CommonAccountInfo {
	log := log15.New("unconfirmedBlocksCache", "toCommonAccountInfo")

	ele := c.blocks.Front()
	var ca CommonAccountInfo
	infoMap := make(map[types.TokenTypeId]*TokenBalanceInfo)
	for ele != nil {

		block := ele.Value.(*ledger.AccountBlock)
		ti, ok := infoMap[block.TokenId]
		if !ok {
			token, err := GetTokenInfoById(&block.TokenId)
			if err != nil {
				log.Error(err.Error())
				continue
			}
			if token == nil {
				log.Error("token nil")
				continue
			}
			infoMap[block.TokenId].Token = *token
			infoMap[block.TokenId].TotalAmount = *block.Amount
			infoMap[block.TokenId].Number = 1
		} else {
			ti.TotalAmount.Add(&ti.TotalAmount, block.Amount)
		}

		ti.Number += 1

		ele = ele.Next()
	}

	ca.TotalNumber = uint64(c.blocks.Len())
	ca.TokenBalanceInfoMap = infoMap
	return &ca
}

func (c *unconfirmedBlocksCache) ResetCursor() {
	c.currentEle = c.blocks.Front()
}

func (c *unconfirmedBlocksCache) GetNextTx() *ledger.AccountBlock {
	if c.currentEle == nil {
		return nil
	}

	block := c.currentEle.Value.(*ledger.AccountBlock)
	c.currentEle = c.currentEle.Next()
	return block
}

func (c *unconfirmedBlocksCache) addTx(b *ledger.AccountBlock) {
	c.blocks.PushBack(b)
}

func (c *unconfirmedBlocksCache) rmTx(b *ledger.AccountBlock) {
	if b == nil {
		return
	}
	ele := c.blocks.Front()
	for ele != nil {
		next := ele.Next()
		if ele.Value.(*ledger.AccountBlock).Hash == b.Hash {
			c.blocks.Remove(ele)
			if ele == c.currentEle {
				c.currentEle = next
			}
		}
		ele = next
	}
}

func AddrListDbSerialize(addrList []*types.Address) ([]byte, error) {
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

func AddrListDbDeserialize(buf []byte) ([]*types.Address, error) {
	var addrListPB = &vitepb.AddressList{}
	if err := proto.Unmarshal(buf, addrListPB); err != nil {
		return nil, err
	}
	var addrList []*types.Address
	for _, addrPB := range addrListPB.AddressList {
		addr, err := types.BytesToAddress(addrPB)
		if err != nil {
			return nil, err
		}
		addrList = append(addrList, &addr)
	}
	return addrList, nil
}
