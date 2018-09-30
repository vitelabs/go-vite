package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"sync"
)

type NeedSnapshotCache struct {
	chain     *chain
	subLedger map[types.Address][]*ledger.AccountBlock

	lock sync.Mutex
	log  log15.Logger
}

func NewNeedSnapshotContent(chain *chain, unconfirmedSubLedger map[types.Address][]*ledger.AccountBlock) *NeedSnapshotCache {
	cache := &NeedSnapshotCache{
		chain:     chain,
		subLedger: unconfirmedSubLedger,
		log:       log15.New("module", "chain/NeedSnapshotCache"),
	}

	return cache
}

func (cache *NeedSnapshotCache) GetSnapshotContent() ledger.SnapshotContent {
	content := make(ledger.SnapshotContent, 0)
	for addr, chain := range cache.subLedger {
		lastBlock := chain[len(chain)-1]
		content[addr] = &ledger.HashHeight{
			Height: lastBlock.Height,
			Hash:   lastBlock.Hash,
		}
	}
	return content
}

func (cache *NeedSnapshotCache) Get(addr *types.Address) []*ledger.AccountBlock {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	return cache.subLedger[*addr]
}
func (cache *NeedSnapshotCache) GetBlockByHashHeight(addr *types.Address, hashHeight *ledger.HashHeight) *ledger.AccountBlock {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	blocks := cache.subLedger[*addr]
	if blocks == nil {
		return nil
	}
	if blocks[0].Height > hashHeight.Height || blocks[len(blocks)-1].Height > hashHeight.Height {
		return nil
	}

	for _, block := range blocks {
		if block.Hash == hashHeight.Hash {
			return block
		}
	}
	return nil
}

func (cache *NeedSnapshotCache) Add(addr *types.Address, accountBlock *ledger.AccountBlock) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	if cachedChain := cache.subLedger[*addr]; len(cachedChain) > 0 && cachedChain[len(cachedChain)-1].Height >= accountBlock.Height {
		cache.log.Error("Can't add", "method", "Add")
		return
	}

	cache.subLedger[*addr] = append(cache.subLedger[*addr], accountBlock)
}

func (cache *NeedSnapshotCache) Remove(addr *types.Address, height uint64) {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	cachedChain := cache.subLedger[*addr]
	if cachedChain == nil {
		cache.log.Error("Can't remove", "method", "Remove")
		return
	}

	var deletedIndex = -1
	for index, item := range cachedChain {
		if item.Height > height {
			break
		}
		deletedIndex = index
	}
	cachedChain = cachedChain[deletedIndex+1:]
	if len(cachedChain) > 0 {
		cache.subLedger[*addr] = cachedChain
	} else {
		delete(cache.subLedger, *addr)
	}
}
