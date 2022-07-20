package onroad_pool

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces/core"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/log15"
)

var initLog = log15.New("initOnRoadPool", nil)

type contractOnRoadPool struct {
	gid     types.Gid
	cache   sync.Map //map[types.Address]*callerCache
	storage *onroadStorage

	chain chainReader
	log   log15.Logger
}

func NewContractOnRoadPool(gid types.Gid, chain chainReader, db *leveldb.DB) OnRoadPool {
	or := &contractOnRoadPool{
		gid:   gid,
		chain: chain,
		log:   log15.New("contractOnRoadPool", gid),

		storage: newOnroadStorage(db),
	}
	if err := or.loadOnRoad(); err != nil {
		panic(fmt.Sprintf("loadOnRoad, err is %v", err))
	}
	return or
}

func (p *contractOnRoadPool) loadOnRoad() error {
	toAddrStat := make(map[types.Address]uint64)
	var lastToAddr *types.Address
	p.log.Info("start loadOnRoad from chain into onroad")
	err := p.chain.LoadOnRoadRange(p.gid, func(fromAddr, toAddr types.Address, hashHeight core.HashHeight) error {
		var cc *callerCache
		if value, ok := p.cache.Load(toAddr); ok {
			cc = value.(*callerCache)
		} else {
			raw, _ := p.cache.LoadOrStore(toAddr, NewCallerCache(toAddr, p.storage))
			cc = raw.(*callerCache)
		}
		if cc == nil {
			return fmt.Errorf("error load caller cache for %s", toAddr)
		}
		if toAddrStat[toAddr] == 0 && lastToAddr != nil {
			p.log.Info(fmt.Sprintf("initLoad one caller, len=%v", toAddrStat[*lastToAddr]), "contract", *lastToAddr)
		}
		toAddrStat[toAddr] = toAddrStat[toAddr] + 1
		lastToAddr = &toAddr
		return cc.initAdd(fromAddr, toAddr, hashHeight)
	})
	if lastToAddr != nil {
		p.log.Info(fmt.Sprintf("initLoad one caller, len=%v", toAddrStat[*lastToAddr]), "contract", *lastToAddr)
	}
	if err != nil {
		return err
	}
	p.log.Info("end loadOnRoad from chain into onroad")
	p.log.Info("success loadOnRoad")
	return nil
}

func (p *contractOnRoadPool) IsFrontOnRoadOfCaller(orAddr, caller types.Address, hash types.Hash) (bool, error) {
	cc, ok := p.cache.Load(orAddr)
	if !ok || cc == nil {
		return false, ErrLoadCallerCacheFailed
	}
	or, err := cc.(*callerCache).getAndLazyUpdateFrontTxByCaller(p.chain, &caller)
	if err != nil {
		return false, err
	}
	if or == nil || or.Hash != hash {
		var frontHash *types.Hash
		if or != nil {
			frontHash = &or.Hash
		}
		p.log.Error(fmt.Sprintf("check IsFrontOnRoadOfCaller fail target=%v front=%v", hash, frontHash))
		return false, ErrCheckIsCallerFrontOnRoadFailed
	}
	return true, nil
}

func (p *contractOnRoadPool) GetFrontOnRoadBlocksByAddr(contract types.Address) ([]*ledger.AccountBlock, error) {
	cc, ok := p.cache.Load(contract)
	if !ok || cc == nil {
		return nil, nil
	}

	blockList := make([]*ledger.AccountBlock, 0)

	orList, err := cc.(*callerCache).getAndLazyUpdateFrontTxOfAllCallers(p.chain)
	if err != nil {
		return nil, err
	}

	for _, or := range orList {
		var err error
		b := or.cachedBlock
		if b == nil {
			b, err = p.chain.GetAccountBlockByHash(or.Hash)
			if err != nil {
				return nil, err
			}
			or.cachedBlock = b
		}
		if b == nil {
			continue
		}
		blockList = append(blockList, b)
	}
	return blockList, nil
}

func (p *contractOnRoadPool) GetOnRoadTotalNumByAddr(contract types.Address) (uint64, error) {
	cc, ok := p.cache.Load(contract)
	if !ok || cc == nil {
		return 0, nil
	}
	return uint64(cc.(*callerCache).len()), nil
}

func (p *contractOnRoadPool) InsertAccountBlocks(orAddr types.Address, blocks []*ledger.AccountBlock) error {
	mlog := p.log.New("method", "InsertAccountBlocks", "orAddr", orAddr, "len", len(blocks))
	isWrite := true
	onroadMap, err := p.ledgerBlockListToOnRoad(orAddr, blocks)
	if err != nil {
		return err
	}
	for _, pendingList := range onroadMap {
		sort.Sort(pendingList)
		for _, v := range pendingList {
			or := v.hashHeight
			if v.block.IsSendBlock() {
				mlog.Debug(fmt.Sprintf("write block-s: %v -> %v %v %v isWrite=%v", v.block.AccountAddress, v.block.ToAddress, v.block.Height, v.block.Hash, isWrite))
				if err := p.insertOnRoad(v.orAddr, v.caller, or, isWrite); err != nil {
					mlog.Error(fmt.Sprintf("write block-s: %v -> %v %v %v isWrite=%v", v.block.AccountAddress, v.block.ToAddress, v.block.Height, v.block.Hash, isWrite),
						"err", err)
					panic("onRoadPool conflict," + err.Error())
				}
			} else {
				mlog.Debug(fmt.Sprintf("write block-r: %v %v %v fromHash=%v isWrite=%v", v.block.AccountAddress, v.block.Height, v.block.Hash, v.block.FromBlockHash, isWrite))
				if err := p.deleteOnRoad(v.orAddr, v.caller, or, isWrite); err != nil {
					mlog.Error(fmt.Sprintf("write block-r: %v %v %v fromHash=%v isWrite=%v", v.block.AccountAddress, v.block.Height, v.block.Hash, v.block.FromBlockHash, isWrite),
						"err", err)
					panic("onRoadPool conflict," + err.Error())
				}
			}
		}
	}
	return nil
}

func (p *contractOnRoadPool) DeleteAccountBlocks(orAddr types.Address, blocks []*ledger.AccountBlock) error {
	mlog := p.log.New("method", "DeleteAccountBlocks", "orAddr", orAddr)
	mlog.Info(fmt.Sprintf("deleteBlocks len %v", len(blocks)))
	isWrite := false
	onroadMap, err := p.ledgerBlockListToOnRoad(orAddr, blocks)
	if err != nil {
		return err
	}
	for _, pendingList := range onroadMap {
		sort.Sort(pendingList)
		for i := pendingList.Len() - 1; i >= 0; i-- {
			v := pendingList[i]
			or := v.hashHeight
			if v.block.IsSendBlock() {
				mlog.Debug(fmt.Sprintf("delete block-s: %v -> %v %v %v isWrite=%v", v.block.AccountAddress, v.block.ToAddress, v.block.Height, v.block.Hash, isWrite))
				if err := p.deleteOnRoad(v.orAddr, v.caller, or, isWrite); err != nil {
					mlog.Error(fmt.Sprintf("delete block-s: %v -> %v %v %v isWrite=%v", v.block.AccountAddress, v.block.ToAddress, v.block.Height, v.block.Hash, isWrite),
						"err", err)
					panic("onRoadPool conflict," + err.Error())
				}
			} else {
				mlog.Debug(fmt.Sprintf("delete block-r: %v %v %v fromHash=%v isWrite=%v", v.block.AccountAddress, v.block.Height, v.block.Hash, v.block.FromBlockHash, isWrite))
				if err := p.insertOnRoad(v.orAddr, v.caller, or, isWrite); err != nil {
					mlog.Error(fmt.Sprintf("delete block-r: %v %v %v fromHash=%v isWrite=%v", v.block.AccountAddress, v.block.Height, v.block.Hash, v.block.FromBlockHash, isWrite),
						"err", err)
					panic("onRoadPool conflict," + err.Error())
				}
			}
		}
	}
	return nil
}

func (p *contractOnRoadPool) insertOnRoad(orAddr, caller types.Address, or orHashHeight, isWrite bool) error {
	p.log.Info(fmt.Sprintf("insert onroad: %s -> %s %d %s isWrite=%v", caller, orAddr, or.Height, or.Hash, isWrite))

	cc, exist := p.cache.Load(orAddr)
	if !exist || cc == nil {
		cc, _ = p.cache.LoadOrStore(orAddr, NewCallerCache(orAddr, p.storage))
	}
	if err := cc.(*callerCache).addTx(&caller, or, isWrite); err != nil {
		return err
	}
	return nil
}

func (p *contractOnRoadPool) deleteOnRoad(orAddr, caller types.Address, or orHashHeight, isWrite bool) error {
	p.log.Info(fmt.Sprintf("delete onroad: %s -> %s %d %s isWrite=%v", caller, orAddr, or.Height, or.Hash, isWrite))
	isCallerContract := types.IsContractAddr(caller)
	cc, exist := p.cache.Load(orAddr)
	if !exist || cc == nil {
		return ErrLoadCallerCacheFailed
	}
	if err := cc.(*callerCache).rmTx(&caller, isCallerContract, or, isWrite); err != nil {
		return err
	}
	return nil
}

func (p *contractOnRoadPool) ledgerBlockListToOnRoad(orAddr types.Address, blocks []*ledger.AccountBlock) (map[types.Address]PendingOnRoadList, error) {
	onroadMap := make(map[types.Address]PendingOnRoadList)
	for _, b := range blocks {
		if b == nil {
			continue
		}
		onroad, err := LedgerBlockToOnRoad(p.chain, b)
		if err != nil {
			if b.IsSendBlock() {
				p.log.Error(fmt.Sprintf("LedgerBlockToOnRoad s fail self=%v t=%v hash=%v height=%v", b.AccountAddress, b.ToAddress, b.Hash, b.Height), "err", err)
			} else {
				p.log.Error(fmt.Sprintf("LedgerBlockToOnRoad r fail self=%v fHash=%v hash=%v height%v", b.AccountAddress, b.FromBlockHash, b.Hash, b.Height), "err", err)
			}
			return nil, err
		}
		_, ok := onroadMap[onroad.caller]
		if !ok {
			onroadMap[onroad.caller] = make(PendingOnRoadList, 0)
		}
		onroadMap[onroad.caller] = append(onroadMap[onroad.caller], onroad)
	}
	return onroadMap, nil
}

func (c *contractOnRoadPool) Info() map[string]interface{} {
	result := make(map[string]interface{})
	sum := 0
	c.cache.Range(func(key, value interface{}) bool {
		len := value.(*callerCache).len()
		result[key.(types.Address).String()] = len
		sum += len
		return true
	})

	result["Sum"] = sum
	return result
}

type callerCache struct {
	storage *onroadStorage
	address types.Address
	mu      sync.RWMutex
}

func NewCallerCache(address types.Address, storage *onroadStorage) *callerCache {
	return &callerCache{
		address: address,
		storage: storage,
	}
}

func (cc *callerCache) initAdd(fromAddr, toAddr types.Address, hashHeight core.HashHeight) error {
	isCallerContract := types.IsContractAddr(fromAddr)
	or := &orHashHeight{
		Height: hashHeight.Height,
		Hash:   hashHeight.Hash,
	}
	if !isCallerContract {
		index := uint32(0)
		or.SubIndex = &index
	}
	initLog.Debug(fmt.Sprintf("addTx %s", or.String()))
	return cc.addTx(&fromAddr, *or, true)
}
func (cc *callerCache) getAndLazyUpdateFrontTxOfAllCallers(reader chainReader) ([]*orHashHeight, error) {
	txs, err := cc.getFrontTxOfAllCallers()
	if err != nil {
		return nil, err
	}

	var result []*orHashHeight

	for _, tx := range txs {
		rr, err := cc.lazyUpdateFrontTx(reader, tx)
		if err != nil {
			onroadPoolLog.Warn("lazy update front tx failed", "err", err)
			continue
		}
		result = append(result, rr)
	}
	return result, nil
}

func (cc *callerCache) getFrontTxOfAllCallers() ([]*orHeightValue, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	m, err := cc.storage.GetAllFirstOnroadTx(cc.address)
	if err != nil {
		return nil, err
	}
	orList := make([]*orHeightValue, 0)
	for _, list := range m {
		front, err2 := newOrHeightValueFromOnroadTxs(list)
		if err2 != nil {
			return nil, err2
		}
		orList = append(orList, &front)
	}
	return orList, nil
}
func (cc *callerCache) getAndLazyUpdateFrontTxByCaller(reader chainReader, caller *types.Address) (*orHashHeight, error) {
	orVal, err := cc.getFrontTxByCaller(caller)
	if err != nil {
		return nil, err
	}
	if orVal == nil {
		return nil, nil
	}

	result, err := cc.lazyUpdateFrontTx(reader, orVal)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (cc *callerCache) getFrontTxByCaller(caller *types.Address) (*orHeightValue, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	txs, err := cc.storage.GetFirstOnroadTx(cc.address, *caller)
	if err != nil {
		return nil, err
	}

	result, err := newOrHeightValueFromOnroadTxs(txs)
	return &result, err
}

func (cc *callerCache) lazyUpdateFrontTx(reader chainReader, hv *orHeightValue) (*orHashHeight, error) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	subs := hv.dirtyTxs()
	if len(subs) > 0 {
		block, err := reader.GetCompleteBlockByHash(subs[0].FromHash)
		if err != nil {
			return nil, err
		}
		if !types.IsContractAddr(block.AccountAddress) {
			return nil, errors.New("get update front tx failed. it's not contract address")
		}
		for i, sendBlock := range block.SendBlockList {
			for _, sub := range subs {
				if sub.FromHash == sendBlock.Hash {
					j := uint32(i)
					sub.FromIndex = &j
					updateResult, err := cc.storage.updateFromIndex(*sub)
					if err != nil {
						return nil, err
					}
					if !updateResult {
						return nil, fmt.Errorf("dirty OnroadTx %s, %s, %s", sub.ToAddr, sub.FromAddr, sub.FromHash)
					}
				}
			}
		}

		subs = hv.dirtyTxs()
		if len(subs) > 0 {
			return nil, errors.New("dirty sub index")
		}
	}
	tx, err := hv.minTx()
	if err != nil {
		return nil, err
	}
	if tx == nil {
		return nil, nil
	}
	return newOrHashHeightFromOnroadTx(tx), nil
}

// @todo fix
func (cc *callerCache) len() int {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	count := 0
	// for _, l := range cc.cache {
	// 	count += l.Len()
	// }
	return count
}

func (cc *callerCache) addTx(caller *types.Address, or orHashHeight, isWrite bool) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return cc.storage.insertOnRoadTx(newOnroadTxFromOrHashHeight(*caller, cc.address, or))
}

func (cc *callerCache) rmTx(caller *types.Address, isCallerContract bool, or orHashHeight, isWrite bool) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	return cc.storage.deleteOnRoadTx(newOnroadTxFromOrHashHeight(*caller, cc.address, or))

}
