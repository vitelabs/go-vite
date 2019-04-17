package chain_plugins

import (
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain/db"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vitepb"
	"math/big"
	"sync"
)

var (
	OneInt = big.NewInt(1)
	oLog   = log15.New("plugin", "onroad_info")
)

type OnRoadInfo struct {
	chain Chain

	store      *chain_db.Store
	storeMutex sync.RWMutex
}

func newOnRoadInfo(store *chain_db.Store, chain Chain) Plugin {
	or := &OnRoadInfo{
		store: store,
		chain: chain,
	}
	if err := or.Clear(); err != nil {
		oLog.Error("onRoadInfo-plugin Clear fail.", "err", err)
		return nil
	}
	oLog.Info("Start InitAndBuild onRoadInfo-plugin")
	if err := or.InitAndBuild(); err != nil {
		oLog.Error("InitAndBuild fail.", "err", err)
		or.Clear()
		return nil
	}
	oLog.Info("InitAndBuild success.")
	return or
}

func (or *OnRoadInfo) Clear() error {
	or.storeMutex.Lock()
	defer or.storeMutex.Unlock()

	iter := or.store.NewIterator(util.BytesPrefix([]byte{OnRoadInfoKeyPrefix}))
	batch := or.store.NewBatch()
	for iter.Next() {
		key := iter.Key()
		or.deleteMeta(batch, key)
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return err
	}
	or.store.WriteDirectly(batch)
	iter.Release()
	return nil
}

func (or *OnRoadInfo) InitAndBuild() error {
	or.storeMutex.Lock()
	defer or.storeMutex.Unlock()

	latestSnapshot := or.chain.GetLatestSnapshotBlock()
	if latestSnapshot == nil {
		return errors.New("GetLatestSnapshotBlock fail.")
	}
	chunks, err := or.chain.GetSubLedger(1, latestSnapshot.Height)
	if err != nil {
		return err
	}
	if len(chunks) <= 0 {
		return nil
	}
	batch := or.store.NewBatch()
	or.writeChunks(batch, chunks)
	or.store.WriteDirectly(batch)

	/*	pendings := excludeWritePairTrades(chunks)
		for _, v := range pendings {
			if v == nil {
				continue
			}
			batch := or.store.NewBatch()
			if err := or.writeBlock(batch, v); err != nil {
				return err
			}
			or.store.WriteDirectly(batch)
		}*/
	return nil
}

func (or *OnRoadInfo) InsertSnapshotBlock(*leveldb.Batch, *ledger.SnapshotBlock, []*ledger.AccountBlock) error {
	return nil
}

func (or *OnRoadInfo) InsertAccountBlock(batch *leveldb.Batch, block *ledger.AccountBlock) error {
	/*	or.storeMutex.Lock()
		defer or.storeMutex.Unlock()

		return or.writeBlock(batch, block)*/
	return nil
}

func (or *OnRoadInfo) DeleteAccountBlocks(*leveldb.Batch, []*ledger.AccountBlock) error {
	return nil
}

func (or *OnRoadInfo) DeleteSnapshotBlocks(batch *leveldb.Batch, chunks []*ledger.SnapshotChunk) error {
	/*	or.storeMutex.Lock()
		defer or.storeMutex.Unlock()
		return or.deleteChunks(batch, chunks)*/
	return nil
}

func (or *OnRoadInfo) GetAccountInfo(addr *types.Address) (*ledger.AccountInfo, error) {
	or.storeMutex.RLock()
	defer or.storeMutex.RUnlock()

	omMap, err := or.readOnRoadInfo(addr)
	if err != nil {
		return nil, err
	}
	onroadInfo := &ledger.AccountInfo{
		AccountAddress:      *addr,
		TotalNumber:         0,
		TokenBalanceInfoMap: make(map[types.TokenTypeId]*ledger.TokenBalanceInfo),
	}
	balanceMap := onroadInfo.TokenBalanceInfoMap
	for k, v := range omMap {
		balanceMap[k] = &ledger.TokenBalanceInfo{
			TotalAmount: v.TotalAmount,
			Number:      v.Number,
		}
		onroadInfo.TotalNumber += v.Number
	}
	return onroadInfo, nil
}

func (or *OnRoadInfo) writeChunks(batch *leveldb.Batch, chunks []*ledger.SnapshotChunk) error {
	for addr, blocks := range excludeWritePairTrades(chunks) {
		oLog.Info(fmt.Sprintf("writeChunks: addr=%v, count=%v", addr, len(blocks)))
		signOmMap, err := or.aggregateBlocks(blocks)
		if err != nil {
			return err
		}
		for tkId, signOm := range signOmMap {
			key := CreateOnRoadInfoKey(&addr, &tkId)
			om, err := or.getMeta(key)
			if err != nil {
				return err
			}
			if om == nil {
				om = &onroadMeta{
					TotalAmount: *big.NewInt(0),
					Number:      0,
				}
			}
			num := new(big.Int).SetUint64(om.Number)
			diffNum := num.Add(num, &signOm.number)
			diffAmount := om.TotalAmount.Add(&om.TotalAmount, &signOm.amount)
			if diffAmount.Sign() < 0 || diffNum.Sign() < 0 || (diffAmount.Sign() > 0 && diffNum.Sign() == 0) {
				return errors.New("conflict, fail to update onroad info")
			}
			if diffNum.Sign() == 0 {
				or.deleteMeta(batch, key)
				continue
			}
			om.TotalAmount = *diffAmount
			om.Number = diffNum.Uint64()
			if err := or.writeMeta(batch, key, om); err != nil {
				return err
			}
		}
	}
	return nil
}

func (or *OnRoadInfo) deleteChunks(batch *leveldb.Batch, chunks []*ledger.SnapshotChunk) error {
	for addr, blocks := range excludeDeletePairTrades(chunks) {
		signOmMap, err := or.aggregateBlocks(blocks)
		if err != nil {
			return err
		}
		for tkId, signOm := range signOmMap {
			key := CreateOnRoadInfoKey(&addr, &tkId)
			om, err := or.getMeta(key)
			if err != nil {
				return err
			}
			if om == nil {
				om = &onroadMeta{
					TotalAmount: *big.NewInt(0),
					Number:      0,
				}
			}
			num := new(big.Int).SetUint64(om.Number)
			diffNum := num.Sub(num, &signOm.number)
			diffAmount := om.TotalAmount.Sub(&om.TotalAmount, &signOm.amount)
			if diffAmount.Sign() < 0 || diffNum.Sign() < 0 || (diffAmount.Sign() > 0 && diffNum.Sign() == 0) {
				return errors.New("conflict, fail to update onroad info")
			}
			if diffNum.Sign() == 0 {
				or.deleteMeta(batch, key)
				continue
			}
			om.TotalAmount = *diffAmount
			om.Number = diffNum.Uint64()
			if err := or.writeMeta(batch, key, om); err != nil {
				return err
			}
		}
	}
	return nil
}

func (or *OnRoadInfo) writeBlock(batch *leveldb.Batch, block *ledger.AccountBlock) error {
	if block.IsSendBlock() {
		key := CreateOnRoadInfoKey(&block.ToAddress, &block.TokenId)
		om, err := or.getMeta(key)
		if err != nil {
			return err
		}
		if om == nil {
			om = &onroadMeta{
				TotalAmount: *big.NewInt(0),
				Number:      0,
			}
		}
		if block.Amount != nil {
			om.TotalAmount.Add(&om.TotalAmount, block.Amount)
		}
		om.Number++
		return or.writeMeta(batch, key, om)
	} else {
		fromBlock, err := or.chain.GetAccountBlockByHash(block.FromBlockHash)
		if err != nil {
			return err
		}
		if fromBlock == nil {
			return errors.New("failed to find onroad by recv")
		}
		key := CreateOnRoadInfoKey(&fromBlock.ToAddress, &fromBlock.TokenId)
		om, err := or.getMeta(key)
		if err != nil {
			return err
		}
		if om == nil {
			return errors.New("conflict, failed to remove onroad cause info meta is nil")
		}
		result := om.TotalAmount.Cmp(fromBlock.Amount)
		if result < 0 || result > 0 && om.Number <= 1 {
			return errors.New("conflict when remove onroad")
		}
		if om.Number <= 1 {
			or.deleteMeta(batch, key)
			return nil
		}
		om.TotalAmount.Sub(&om.TotalAmount, fromBlock.Amount)
		om.Number--
		return or.writeMeta(batch, key, om)
	}
}

func (or *OnRoadInfo) deleteBlock(batch *leveldb.Batch, block *ledger.AccountBlock) error {
	//fmt.Printf("block: addr=%v hash=%v height=%v\n", block.AccountAddress, block.Hash, block.Height)
	if block.IsSendBlock() {
		key := CreateOnRoadInfoKey(&block.ToAddress, &block.TokenId)
		om, err := or.getMeta(key)
		if err != nil {
			return err
		}
		if om == nil {
			return errors.New("conflict, failed to remove onroad cause info meta is nil")
		}
		if om.TotalAmount.Cmp(block.Amount) == -1 {
			return errors.New("conflict with amount of onroad info")
		} else if om.TotalAmount.Cmp(block.Amount) == 0 {
			or.deleteMeta(batch, key)
			return nil
		} else {
			om.TotalAmount.Sub(&om.TotalAmount, block.Amount)
			om.Number--
			return or.writeMeta(batch, key, om)
		}
	} else {
		fromBlock, err := or.chain.GetAccountBlockByHash(block.FromBlockHash)
		if err != nil {
			return err
		}
		if fromBlock == nil {
			return errors.New("failed to find onroad by recv")
		}
		//fmt.Printf("fromBlock: addr=%v hash=%v height=%v\n", fromBlock.AccountAddress, fromBlock.Hash, fromBlock.Height)
		key := CreateOnRoadInfoKey(&fromBlock.ToAddress, &fromBlock.TokenId)
		om, err := or.getMeta(key)
		if err != nil {
			return err
		}
		if om == nil {
			om = &onroadMeta{
				TotalAmount: *big.NewInt(0),
				Number:      0,
			}
		}
		if fromBlock.Amount != nil {
			om.TotalAmount.Add(&om.TotalAmount, fromBlock.Amount)
		}
		om.Number++
		return or.writeMeta(batch, key, om)
	}
}

type signOnRoadMeta struct {
	amount big.Int
	number big.Int
}

func (or *OnRoadInfo) aggregateBlocks(blocks []*ledger.AccountBlock) (map[types.TokenTypeId]*signOnRoadMeta, error) {
	addMap := make(map[types.TokenTypeId]*signOnRoadMeta)
	for _, block := range blocks {
		if block.IsSendBlock() {
			v, ok := addMap[block.TokenId]
			if !ok || v == nil {
				v = &signOnRoadMeta{
					amount: *big.NewInt(0),
					number: *big.NewInt(0),
				}
			}
			if block.Amount != nil {
				v.amount.Add(&v.amount, block.Amount)
			}
			v.number.Add(&v.number, OneInt)
			addMap[block.TokenId] = v
		} else {
			fromBlock, err := or.chain.GetAccountBlockByHash(block.FromBlockHash)
			if err != nil {
				return nil, err
			}
			if fromBlock == nil {
				return nil, errors.New("failed to find onroad by recv")
			}
			v, ok := addMap[fromBlock.TokenId]
			if !ok || v == nil {
				v = &signOnRoadMeta{
					amount: *big.NewInt(0),
					number: *big.NewInt(0),
				}
			}
			if fromBlock.Amount != nil {
				v.amount.Sub(&v.amount, fromBlock.Amount)
			}
			v.number.Sub(&v.number, OneInt)
			addMap[fromBlock.TokenId] = v
		}
	}
	return addMap, nil
}

func (or *OnRoadInfo) readOnRoadInfo(addr *types.Address) (map[types.TokenTypeId]*onroadMeta, error) {
	omMap := make(map[types.TokenTypeId]*onroadMeta)
	iter := or.store.NewIterator(util.BytesPrefix(CreateOnRoadInfoPrefixKey(addr)))
	defer iter.Release()

	for iter.Next() {
		key := iter.Key()
		tokenTypeIdBytes := key[1+types.AddressSize : 1+types.AddressSize+types.TokenTypeIdSize]
		tokenTypeId, err := types.BytesToTokenTypeId(tokenTypeIdBytes)
		if err != nil {
			return nil, err
		}
		om := &onroadMeta{}
		if err := om.deserialize(iter.Value()); err != nil {
			return nil, err
		}
		omMap[tokenTypeId] = om
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}
	return omMap, nil
}

func (or *OnRoadInfo) getMeta(key []byte) (*onroadMeta, error) {
	value, err := or.store.Get(key)
	if err != nil {
		return nil, err
	}
	if len(value) <= 0 {
		return nil, nil
	}
	om := &onroadMeta{}
	if err := om.deserialize(value); err != nil {
		return nil, err
	}
	return om, nil
}

func (or *OnRoadInfo) writeMeta(batch *leveldb.Batch, key []byte, meta *onroadMeta) error {
	dataSlice, sErr := meta.serialize()
	if sErr != nil {
		return sErr
	}
	batch.Put(key, dataSlice)
	return nil
}

func (or *OnRoadInfo) deleteMeta(batch *leveldb.Batch, key []byte) {
	batch.Delete(key)
}

type onroadMeta struct {
	TotalAmount big.Int
	Number      uint64
}

func (om *onroadMeta) proto() *vitepb.OnroadMeta {
	pb := &vitepb.OnroadMeta{}
	pb.Num = om.Number
	pb.Amount = om.TotalAmount.Bytes()
	return pb
}

func (om *onroadMeta) deProto(pb *vitepb.OnroadMeta) {
	om.Number = pb.Num
	totalAmount := big.NewInt(0)
	if len(pb.Amount) > 0 {
		totalAmount.SetBytes(pb.Amount)
	}
	om.TotalAmount = *totalAmount
}

func (om *onroadMeta) serialize() ([]byte, error) {
	return proto.Marshal(om.proto())
}

func (om *onroadMeta) deserialize(buf []byte) error {
	pb := &vitepb.OnroadMeta{}
	if err := proto.Unmarshal(buf, pb); err != nil {
		return err
	}
	om.deProto(pb)
	return nil
}

func CreateOnRoadInfoKey(addr *types.Address, tId *types.TokenTypeId) []byte {
	key := make([]byte, 0, 1+types.AddressSize+types.TokenTypeIdSize)
	key = append(key, OnRoadInfoKeyPrefix)
	key = append(key, addr.Bytes()...)
	key = append(key, tId.Bytes()...)
	return key
}

func CreateOnRoadInfoPrefixKey(addr *types.Address) []byte {
	key := make([]byte, 0, 1+types.AddressSize)
	key = append(key, OnRoadInfoKeyPrefix)
	key = append(key, addr.Bytes()...)
	return key
}

func excludeWritePairTrades(chunks []*ledger.SnapshotChunk) map[types.Address][]*ledger.AccountBlock {
	cutMap := make(map[types.Hash]*ledger.AccountBlock)
	for _, chunk := range chunks {
		for _, p := range chunk.AccountBlocks {
			if p.IsSendBlock() {
				cutMap[p.Hash] = p
				continue
			}

			v, ok := cutMap[p.FromBlockHash]
			if ok && v != nil && v.IsSendBlock() {
				delete(cutMap, p.FromBlockHash)
				continue
			}
			cutMap[p.FromBlockHash] = p
		}
	}
	pendingMap := make(map[types.Address][]*ledger.AccountBlock)
	for _, v := range cutMap {
		if v == nil {
			continue
		}
		var addr *types.Address
		if v.IsSendBlock() {
			addr = &v.ToAddress
		} else {
			addr = &v.AccountAddress
		}
		_, ok := pendingMap[*addr]
		if !ok {
			list := make([]*ledger.AccountBlock, 0)
			list = append(list, v)
			pendingMap[*addr] = list
		} else {
			pendingMap[*addr] = append(pendingMap[*addr], v)
		}
	}
	return pendingMap
}

func excludeDeletePairTrades(chunks []*ledger.SnapshotChunk) map[types.Address][]*ledger.AccountBlock {
	cutMap := make(map[types.Hash]*ledger.AccountBlock)
	for i := len(chunks) - 1; i >= 0; i-- {
		for _, p := range chunks[i].AccountBlocks {
			if p.IsSendBlock() {
				v, ok := cutMap[p.Hash]
				if ok && v != nil && v.IsReceiveBlock() {
					delete(cutMap, p.Hash)
					continue
				}
				cutMap[p.Hash] = p
				continue
			}

			v, ok := cutMap[p.FromBlockHash]
			if ok && v != nil && v.IsSendBlock() {
				delete(cutMap, p.FromBlockHash)
				continue
			}
			cutMap[p.FromBlockHash] = p
		}
	}

	pendingMap := make(map[types.Address][]*ledger.AccountBlock)
	for _, v := range cutMap {
		if v == nil {
			continue
		}
		var addr *types.Address
		if v.IsSendBlock() {
			addr = &v.ToAddress
		} else {
			addr = &v.AccountAddress
		}
		_, ok := pendingMap[*addr]
		if !ok {
			list := make([]*ledger.AccountBlock, 0)
			list = append(list, v)
			pendingMap[*addr] = list
		} else {
			pendingMap[*addr] = append(pendingMap[*addr], v)
		}
	}
	return pendingMap
}
