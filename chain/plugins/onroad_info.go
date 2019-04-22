package chain_plugins

import (
	"github.com/go-errors/errors"
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

	/*	index        = 0
		exlcudeIndex = 0
		insertSb     = false
		insertAb     = false
		bugHash, _   = types.HexToHash("dd356e3190a1eaace06f5131ba1107436035a1c605467c311b4d6d9935416ed3")*/
)

type OnRoadInfo struct {
	chain Chain

	unconfirmedCache map[types.Address]map[types.Hash]*ledger.AccountBlock
	store            *chain_db.Store
	mu               sync.RWMutex
}

func newOnRoadInfo(store *chain_db.Store, chain Chain) Plugin {
	or := &OnRoadInfo{
		chain: chain,

		store:            store,
		unconfirmedCache: make(map[types.Address]map[types.Hash]*ledger.AccountBlock),
	}
	return or
}

func (or *OnRoadInfo) InitAndBuild() error {
	// fixme
	return nil
}

func (or *OnRoadInfo) Clear() error {
	// clean cache
	or.unconfirmedCache = make(map[types.Address]map[types.Hash]*ledger.AccountBlock)

	// clean db
	/*	iter := or.store.NewIterator(util.BytesPrefix([]byte{OnRoadInfoKeyPrefix}))
		batch := or.store.NewBatch()
		for iter.Next() {
			key := iter.Key()
			or.deleteMeta(batch, key)
		}
		if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
			return err
		}
		or.store.WriteDirectly(batch)
		iter.Release()*/
	return nil
}

func (or *OnRoadInfo) InsertSnapshotBlock(batch *leveldb.Batch, snapshotBlock *ledger.SnapshotBlock, confirmedBlocks []*ledger.AccountBlock) error {
	/*	insertSb = true
		defer func() { insertSb = false }()*/
	addrOnRoadMap, err := excludePairTrades(or.chain, confirmedBlocks)
	if err != nil {
		return err
	}

	or.mu.Lock()
	defer or.mu.Unlock()

	or.removeUnconfirmed(addrOnRoadMap)

	return or.flushWriteBySnapshotLine(batch, addrOnRoadMap)
}

func (or *OnRoadInfo) DeleteSnapshotBlocks(batch *leveldb.Batch, chunks []*ledger.SnapshotChunk) error {
	if len(chunks) <= 0 {
		return nil
	}

	blocks := make([]*ledger.AccountBlock, 0)
	for i := len(chunks) - 1; i >= 0; i-- {
		if len(chunks[i].AccountBlocks) <= 0 {
			continue
		}
		blocks = append(blocks, chunks[i].AccountBlocks...)
	}
	addrOnRoadMap, err := excludePairTrades(or.chain, blocks)
	if err != nil {
		return err
	}

	or.mu.Lock()
	defer or.mu.Unlock()

	// clean unconfirmed cache
	or.unconfirmedCache = make(map[types.Address]map[types.Hash]*ledger.AccountBlock)

	// revert flush the db
	return or.flushDeleteBySnapshotLine(batch, addrOnRoadMap)
}

func (or *OnRoadInfo) InsertAccountBlock(batch *leveldb.Batch, block *ledger.AccountBlock) error {
	/*	insertAb = true
		defer func() { insertAb = false }()*/

	blocks := make([]*ledger.AccountBlock, 0)
	blocks = append(blocks, block)
	addrOnRoadMap, err := excludePairTrades(or.chain, blocks)
	if err != nil {
		return err
	}

	or.mu.Lock()
	defer or.mu.Unlock()

	or.addUnconfirmed(addrOnRoadMap)
	return nil
}

func (or *OnRoadInfo) DeleteAccountBlocks(batch *leveldb.Batch, blocks []*ledger.AccountBlock) error {
	addrOnRoadMap, err := excludePairTrades(or.chain, blocks)
	if err != nil {
		return err
	}

	or.mu.Lock()
	defer or.mu.Unlock()

	or.removeUnconfirmed(addrOnRoadMap)
	return nil
}

func (or *OnRoadInfo) GetAccountInfo(addr *types.Address) (*ledger.AccountInfo, error) {
	if addr == nil {
		return nil, nil
	}

	or.mu.RLock()
	defer or.mu.RUnlock()

	omMap, err := or.readOnRoadInfo(addr)
	if err != nil {
		return nil, err
	}

	signOmMap, err := or.getUnconfirmed(*addr)
	if err != nil {
		return nil, err
	}
	for tkId, signOm := range signOmMap {
		om, ok := omMap[tkId]
		if !ok || om == nil {
			om = &onroadMeta{
				TotalAmount: *big.NewInt(0),
				Number:      0,
			}
		}
		num := new(big.Int).SetUint64(om.Number)
		diffNum := num.Add(num, &signOm.number)
		diffAmount := om.TotalAmount.Add(&om.TotalAmount, &signOm.amount)
		if diffAmount.Sign() < 0 || diffNum.Sign() < 0 || (diffAmount.Sign() > 0 && diffNum.Sign() == 0) {
			return nil, errors.New("conflict, fail to update onroad info")
		}
		if diffNum.Sign() == 0 {
			delete(omMap, tkId)
			continue
		}
		om.TotalAmount = *diffAmount
		om.Number = diffNum.Uint64()
		omMap[tkId] = om
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

func (or *OnRoadInfo) getUnconfirmed(addr types.Address) (map[types.TokenTypeId]*signOnRoadMeta, error) {
	onRoadMap, ok := or.unconfirmedCache[addr]
	if !ok || onRoadMap == nil {
		return nil, nil
	}
	pendingMap := make([]*ledger.AccountBlock, 0)
	for _, v := range onRoadMap {
		if v == nil {
			continue
		}
		pendingMap = append(pendingMap, v)
	}
	return or.aggregateBlocks(pendingMap)
}

func (or *OnRoadInfo) addUnconfirmed(addrMap map[types.Address][]*ledger.AccountBlock) {
	for addr, blockList := range addrMap {
		if len(blockList) <= 0 {
			continue
		}
		onRoadMap, ok := or.unconfirmedCache[addr]
		if !ok || onRoadMap == nil {
			onRoadMap = make(map[types.Hash]*ledger.AccountBlock)
		}
		for _, block := range blockList {
			var hashKey types.Hash
			if block.IsSendBlock() {
				hashKey = block.Hash
			} else {
				hashKey = block.FromBlockHash
			}
			value := onRoadMap[hashKey]
			if value != nil && value.IsSendBlock() != block.IsSendBlock() {
				delete(onRoadMap, hashKey)
			} else {
				onRoadMap[hashKey] = block
			}
			or.unconfirmedCache[addr] = onRoadMap
		}
	}
}

func (or *OnRoadInfo) removeUnconfirmed(addrMap map[types.Address][]*ledger.AccountBlock) {
	for addr, blockList := range addrMap {
		if len(blockList) <= 0 {
			continue
		}
		onRoadMap, ok := or.unconfirmedCache[addr]
		if !ok || onRoadMap == nil {
			continue
		}
		for _, block := range blockList {
			var hashKey types.Hash
			if block.IsSendBlock() {
				hashKey = block.Hash
			} else {
				hashKey = block.FromBlockHash
			}
			value := onRoadMap[hashKey]
			if value != nil && value.IsSendBlock() == block.IsSendBlock() {
				delete(onRoadMap, hashKey)
			}
		}
	}
}

func (or *OnRoadInfo) flushWriteBySnapshotLine(batch *leveldb.Batch, confirmedBlocks map[types.Address][]*ledger.AccountBlock) error {
	for addr, pendingList := range confirmedBlocks {
		signOmMap, err := or.aggregateBlocks(pendingList)
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

			/*			if diffAmount.Sign() != 0 {
						//fmt.Printf("index=%v addr=%v tk=%v diff[%v %v] hash:\n", index, addr, tkId, diffNum.String(), diffAmount.String())
						for _, v := range pendingList {
							if v.IsReceiveBlock() {
								fromBlock, _ := or.chain.GetAccountBlockByHash(v.FromBlockHash)
								fmt.Printf("-- Recv addr=%v hash=%v amount=%v recvHash=%v\n", addr, fromBlock.Hash, fromBlock.Amount, v.Hash)
							} else {
								fmt.Printf("-- Send addr=%v hash=%v amount=%v\n", addr, v.Hash, v.Amount)
							}
						}
					}*/

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

func (or *OnRoadInfo) flushDeleteBySnapshotLine(batch *leveldb.Batch, confirmedBlocks map[types.Address][]*ledger.AccountBlock) error {
	for addr, pendingList := range confirmedBlocks {
		signOmMap, err := or.aggregateBlocks(pendingList)
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

func excludePairTrades(chain Chain, blockList []*ledger.AccountBlock) (map[types.Address][]*ledger.AccountBlock, error) {
	/*	defer func() { exlcudeIndex++ }()*/

	cutMap := make(map[types.Hash]*ledger.AccountBlock)
	for _, block := range blockList {
		if block.IsSendBlock() {
			v, ok := cutMap[block.Hash]
			if ok && v != nil && v.IsReceiveBlock() {
				delete(cutMap, block.Hash)
			} else {
				cutMap[block.Hash] = block
			}
			continue
		}

		if chain.IsGenesisAccountBlock(block.Hash) {
			continue
		}

		v, ok := cutMap[block.FromBlockHash]
		if ok && v != nil && v.IsSendBlock() {
			delete(cutMap, block.FromBlockHash)
			continue
		}
		cutMap[block.FromBlockHash] = block

		// sendBlockList
		isContract, err := chain.IsContractAccount(block.AccountAddress)
		if err != nil {
			return nil, err
		}
		if !isContract || len(block.SendBlockList) <= 0 {
			continue
		}
		for _, subSend := range block.SendBlockList {
			v, ok := cutMap[subSend.Hash]
			if ok && v != nil && v.IsReceiveBlock() {
				delete(cutMap, subSend.Hash)
			} else {
				cutMap[subSend.Hash] = subSend
			}
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
		/*		if bugHash == v.Hash {
				fmt.Printf("%v %v %v insertAb=%v insertSb=%v\n", exlcudeIndex, hash, v.FromBlockHash, insertAb, insertSb)
			}*/
		_, ok := pendingMap[*addr]
		if !ok {
			list := make([]*ledger.AccountBlock, 0)
			list = append(list, v)
			pendingMap[*addr] = list
		} else {
			pendingMap[*addr] = append(pendingMap[*addr], v)
		}
	}
	return pendingMap, nil
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
