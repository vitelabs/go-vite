package chain_cache

import (
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vitepb"
	"sync"
	"time"
)

type Fragment struct {
	HeadHeight uint64
	TailHeight uint64
	List       []*AdditionItem
}

func (frag *Fragment) GetDbKey() []byte {
	tailHeightKey := make([]byte, 8)
	binary.BigEndian.PutUint64(tailHeightKey, frag.TailHeight)

	headHeightKey := make([]byte, 8)
	binary.BigEndian.PutUint64(headHeightKey, frag.HeadHeight)

	key, _ := database.EncodeKey(database.DBKP_ADDITIONAL_LIST, tailHeightKey, headHeightKey)
	return key
}

func GetFragTailHeightFromDbKey(dbKey []byte) uint64 {
	return binary.BigEndian.Uint64(dbKey[1:9])
}

func GetFragHeadHeightFromDbKey(dbKey []byte) uint64 {
	return binary.BigEndian.Uint64(dbKey[9:17])
}

func (frag *Fragment) Serialize() ([]byte, error) {
	listPb := make([]*vitepb.SnapshotAdditionalItem, len(frag.List))

	for index, fragAdditionItem := range frag.List {
		listPb[index] = fragAdditionItem.Proto()
	}

	pb := &vitepb.SnapshotAdditionalFragment{
		List: listPb,
	}
	return proto.Marshal(pb)
}

func (frag *Fragment) Deserialize(buf []byte) error {
	pb := &vitepb.SnapshotAdditionalFragment{}
	if err := proto.Unmarshal(buf, pb); err != nil {
		return err
	}

	frag.List = make([]*AdditionItem, len(pb.List))
	for index, pbItem := range pb.List {
		ai := &AdditionItem{}
		ai.DeProto(pbItem)
		frag.List[index] = ai
	}
	return nil
}

type AdditionItem struct {
	Quota              uint64
	AggregateQuota     uint64
	SnapshotHashHeight *ledger.HashHeight
}

func (ai *AdditionItem) DeProto(pb *vitepb.SnapshotAdditionalItem) {
	ai.Quota = pb.Quota
	ai.AggregateQuota = pb.AggregateQuota
	hash, _ := types.BytesToHash(pb.SnapshotHashHeight.Hash)
	ai.SnapshotHashHeight = &ledger.HashHeight{
		Hash:   hash,
		Height: pb.SnapshotHashHeight.Height,
	}
}

func (ai *AdditionItem) Proto() *vitepb.SnapshotAdditionalItem {
	hashHeightPb := &vitepb.HashHeight{
		Hash:   ai.SnapshotHashHeight.Hash.Bytes(),
		Height: ai.SnapshotHashHeight.Height,
	}

	pb := &vitepb.SnapshotAdditionalItem{
		Quota:              ai.Quota,
		AggregateQuota:     ai.AggregateQuota,
		SnapshotHashHeight: hashHeightPb,
	}
	return pb
}

type AdditionList struct {
	list  []*AdditionItem
	frags []*Fragment

	aggregateHeight uint64
	saveHeight      int
	flushInterval   time.Duration

	chain Chain

	log log15.Logger

	modifyLock sync.RWMutex
	wg         sync.WaitGroup
	statusLock sync.Mutex
	status     int // 0 means stop, 1 means start
	timer      *time.Ticker
	terminal   chan struct{}
}

func NewAdditionList(chain Chain) (*AdditionList, error) {
	al := &AdditionList{
		aggregateHeight: 60 * 60,
		saveHeight:      5 * 24 * 60 * 60,

		flushInterval: time.Hour * 1,
		status:        0,
		log:           log15.New("module", "snapshot_additional_list"),
		chain:         chain,
	}

	if err := al.loadFromDb(); err != nil {
		al.log.Error("al.loadFromDb failed, error is "+err.Error(), "method", "NewAdditionList")
		return nil, err
	}

	al.log.Info("Calculate the entire network quota and build cache...")
	if err := al.build(); err != nil {
		al.log.Error("al.build failed, error is "+err.Error(), "method", "NewAdditionList")
		return nil, err
	}
	al.log.Info("Complete the calculation of entire network quota and cache build")

	al.flush()

	return al, nil
}

func (al *AdditionList) Start() {
	al.statusLock.Lock()
	defer al.statusLock.Unlock()
	if al.status == 1 {
		return
	}
	al.terminal = make(chan struct{})

	al.wg.Add(1)
	go func() {
		defer al.wg.Done()
		al.timer = time.NewTicker(al.flushInterval)
		for {
			select {
			case <-al.timer.C:
				al.flush()
			case <-al.terminal:
				return
			}
		}

	}()
	al.status = 1
}

func (al *AdditionList) Stop() {
	al.statusLock.Lock()
	defer al.statusLock.Unlock()
	if al.status == 0 {
		return
	}

	al.timer.Stop()
	close(al.terminal)
	al.wg.Wait()
	al.status = 0
}

func (al *AdditionList) build() error {
	al.modifyLock.Lock()
	defer al.modifyLock.Unlock()

	latestSnapshotBlock := al.chain.GetLatestSnapshotBlock()
	if latestSnapshotBlock == nil {
		return nil
	}
	latestHeight := latestSnapshotBlock.Height

	// prepend
	prependTailHeight := uint64(1)
	if latestHeight > uint64(al.saveHeight) {
		prependTailHeight = latestHeight - uint64(al.saveHeight) + 1
	}

	prependHeadHeight := latestHeight

	if len(al.list) > 0 {
		prependHeadHeight = al.list[0].SnapshotHashHeight.Height - 1
	}

	if prependTailHeight < prependHeadHeight {
		needBuildPrependTailHeight := uint64(1)
		if prependTailHeight > al.aggregateHeight {
			needBuildPrependTailHeight = prependTailHeight - al.aggregateHeight + 1
		}
		count := prependHeadHeight - needBuildPrependTailHeight + 1
		snapshotBlocks, err := al.chain.GetSnapshotBlocksByHeight(needBuildPrependTailHeight, count, true, true)
		if err != nil {
			return err
		}

		originList := al.list
		al.list = make([]*AdditionItem, 0)
		if err := al.addList(snapshotBlocks); err != nil {
			return err
		}

		al.list = append(al.list, originList...)
	}

	// append
	appendTailHeight := uint64(0)
	appendHeadHeight := uint64(0)
	if len(al.list) > 0 {
		appendTailHeight = al.list[len(al.list)-1].SnapshotHashHeight.Height + 1
		if appendTailHeight <= latestHeight {
			appendHeadHeight = latestHeight
		}
	}

	if appendTailHeight > 0 &&
		appendHeadHeight > 0 &&
		appendHeadHeight >= appendTailHeight {

		count := appendHeadHeight - appendTailHeight + 1
		snapshotBlocks, err := al.chain.GetSnapshotBlocksByHeight(appendTailHeight, count, true, true)
		if err != nil {
			return err
		}
		al.addList(snapshotBlocks)
	}
	return nil
}

func (al *AdditionList) flush() {
	al.modifyLock.Lock()
	defer al.modifyLock.Unlock()

	if len(al.list) <= 0 {
		return
	}

	headAdditionItem := al.list[len(al.list)-1]
	tailAdditionItem := al.list[0]

	NewFragTailHeight := tailAdditionItem.SnapshotHashHeight.Height
	if len(al.frags) > 0 {
		savedFragHeadHeight := al.frags[len(al.frags)-1].HeadHeight

		NewFragTailHeight = savedFragHeadHeight + 1
	}
	NewFragHeadHeight := headAdditionItem.SnapshotHashHeight.Height

	if NewFragTailHeight > NewFragHeadHeight {
		return
	}

	newFragListStartIndex := al.getIndexByHeight(NewFragTailHeight)
	newFragListEndIndex := al.getIndexByHeight(NewFragHeadHeight)

	newFragList := al.list[newFragListStartIndex : newFragListEndIndex+1]

	newFrag := &Fragment{
		HeadHeight: NewFragHeadHeight,
		TailHeight: NewFragTailHeight,
		List:       newFragList,
	}

	if err := al.saveFrag(newFrag); err != nil {
		al.log.Error("saveFrag failed, error is "+err.Error(), "method", "flush")
		return
	}
	al.frags = append(al.frags, newFrag)
	al.clearStaleData()
}

func (al *AdditionList) clearStaleData() {
	count := len(al.list)
	if count <= 0 {
		return
	}

	if count <= al.saveHeight {
		return
	}

	needClearCount := count - al.saveHeight
	needClearAdditionItem := al.list[needClearCount-1]

	needClearFrags := make([]*Fragment, 0)
	needClearIndex := 0

	for index, frag := range al.frags {
		if frag.HeadHeight <= needClearAdditionItem.SnapshotHashHeight.Height {
			needClearFrags = append(needClearFrags, frag)
			needClearIndex = index
		}
	}

	if len(needClearFrags) <= 0 {
		return
	}

	if err := al.deleteFrags(needClearFrags); err != nil {
		al.log.Error("deleteFrags failed, error is "+err.Error(), "method", "clearStaleData")
		return
	}
	al.frags = al.frags[needClearIndex+1:]
	al.list = al.list[needClearCount:]
}

func (al *AdditionList) deleteFrags(fragments []*Fragment) error {
	batch := new(leveldb.Batch)

	for _, fragment := range fragments {
		key := fragment.GetDbKey()
		batch.Delete(key)
	}

	return al.chain.ChainDb().Commit(batch)
}

func (al *AdditionList) saveFrag(fragment *Fragment) error {
	key := fragment.GetDbKey()
	value, err := fragment.Serialize()
	if err != nil {
		return err
	}

	db := al.chain.ChainDb().Db()
	if err := db.Put(key, value, nil); err != nil {
		return err
	}

	return nil
}

func (al *AdditionList) loadFromDb() error {
	db := al.chain.ChainDb().Db()

	iter := db.NewIterator(util.BytesPrefix([]byte{database.DBKP_ADDITIONAL_LIST}), nil)
	defer iter.Release()

	var frags []*Fragment
	var list []*AdditionItem

	for iter.Next() {
		value := iter.Value()
		frag := &Fragment{}
		if err := frag.Deserialize(value); err != nil {
			return err
		}
		frag.TailHeight = GetFragTailHeightFromDbKey(iter.Key())
		frag.HeadHeight = GetFragHeadHeightFromDbKey(iter.Key())

		frags = append(frags, frag)
		list = append(list, frag.List...)
	}
	if err := iter.Error(); err != nil &&
		err != leveldb.ErrNotFound {
		return err
	}

	al.frags = frags
	al.list = list

	return nil
}

func (al *AdditionList) addList(snapshotBlocks []*ledger.SnapshotBlock) error {
	for _, snapshotBlock := range snapshotBlocks {
		subLedger, err := al.chain.GetConfirmSubLedgerBySnapshotBlocks([]*ledger.SnapshotBlock{snapshotBlock})
		if err != nil {
			return err
		}

		quota := uint64(0)
		for _, blocks := range subLedger {
			for _, block := range blocks {
				quota += block.Quota
			}
		}
		al.add(snapshotBlock, quota)
	}
	return nil
}

func (al *AdditionList) add(block *ledger.SnapshotBlock, quota uint64) {
	aggregateQuota := al.calculateQuota(block, quota)
	ai := &AdditionItem{
		Quota:          quota,
		AggregateQuota: aggregateQuota,
		SnapshotHashHeight: &ledger.HashHeight{
			Hash:   block.Hash,
			Height: block.Height,
		},
	}
	al.list = append(al.list, ai)
}

func (al *AdditionList) Add(block *ledger.SnapshotBlock, quota uint64) {
	al.modifyLock.Lock()
	defer al.modifyLock.Unlock()

	al.add(block, quota)
}

func (al *AdditionList) DeleteStartWith(block *ledger.SnapshotBlock) error {
	al.modifyLock.Lock()
	defer al.modifyLock.Unlock()

	index := al.getIndexByHeight(block.Height)
	if index < 0 {
		return errors.New(fmt.Sprintf("Can't find block, block hash is %s, block height is %d", block.Hash, block.Height))
	}
	ai := al.list[index]
	if ai.SnapshotHashHeight.Hash != block.Hash {
		return errors.New(fmt.Sprintf("Block hash is error, block hash is %s, block height is %d", block.Hash, block.Height))
	}
	al.list = al.list[:index]

	return nil
}

func (al *AdditionList) GetAggregateQuota(block *ledger.SnapshotBlock) (uint64, error) {
	al.modifyLock.RLock()
	defer al.modifyLock.RUnlock()

	item := al.getByHeight(block.Height)
	if item == nil || item.SnapshotHashHeight.Hash != block.Hash {
		err := errors.New(fmt.Sprintf("hash %s not found.", block.Hash))
		return 0, err
	}
	return item.AggregateQuota, nil
}

func (al *AdditionList) calculateQuota(block *ledger.SnapshotBlock, quota uint64) uint64 {
	if block.Height <= 1 {
		return quota
	}

	prevHeight := block.Height - 1
	prevAdditionItem := al.getByHeight(prevHeight)

	if prevAdditionItem == nil {
		return quota
	}

	if block.Height <= al.aggregateHeight {

		aggregateQuota := prevAdditionItem.AggregateQuota + quota
		return aggregateQuota
	}

	tailAdditionItem := al.getByHeight(prevHeight - al.aggregateHeight + 1)
	if tailAdditionItem == nil {
		return prevAdditionItem.AggregateQuota + quota
	}

	aggregateQuota := prevAdditionItem.AggregateQuota - tailAdditionItem.Quota + quota
	return aggregateQuota
}

func (al *AdditionList) getIndexByHeight(height uint64) int {
	if len(al.list) <= 0 {
		return -1
	}
	headAdditionItem := al.list[len(al.list)-1]
	tailAdditionItem := al.list[0]

	headHeight := headAdditionItem.SnapshotHashHeight.Height
	if headHeight < height {
		return -1
	}

	tailHeight := tailAdditionItem.SnapshotHashHeight.Height
	if tailHeight > height {
		return -1
	}

	index := int(height - tailHeight)
	return index

}
func (al *AdditionList) getByHeight(height uint64) *AdditionItem {
	index := al.getIndexByHeight(height)
	if index < 0 {
		return nil
	}
	return al.list[index]
}
