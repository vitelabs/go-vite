package pool

import (
	"testing"
	"time"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
)

type mockCommonBlock struct {
	height   uint64
	hash     types.Hash
	prevHash types.Hash
}

func (self *mockCommonBlock) ReferHashes() ([]types.Hash, []types.Hash, *types.Hash) {
	panic("implement me")
}

func (self *mockCommonBlock) Source() types.BlockSource {
	panic("implement me")
}

func newMockCommonBlock(s int, i uint64) *mockCommonBlock {
	r := &mockCommonBlock{}
	r.height = i
	r.hash = common.MockHashBy(s, int(i))
	r.prevHash = common.MockHashBy(s, int(i-1))
	return r
}

func (self *mockCommonBlock) Height() uint64 {
	return self.height
}

func (self *mockCommonBlock) Hash() types.Hash {
	return self.hash
}

func (self *mockCommonBlock) PrevHash() types.Hash {
	return self.prevHash
}

func (*mockCommonBlock) checkForkVersion() bool {
	panic("implement me")
}

func (*mockCommonBlock) resetForkVersion() {
	panic("implement me")
}

func (*mockCommonBlock) forkVersion() int {
	panic("implement me")
}

//type mockSnapshotI interface {
//	sverifier
//	syncer
//	chainDb
//	chainRw
//}
//
//type mockSnapshotS struct {
//}
//
//func (*mockSnapshotS) GetSnapshotBlockHeadByHeight(height uint64) (*ledger.SnapshotBlock, error) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) IsGenesisSnapshotBlock(block *ledger.SnapshotBlock) bool {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) IsGenesisAccountBlock(block *ledger.AccountBlock) bool {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) FetchAccountBlocksWithHeight(start types.Hash, count uint64, address *types.Address, sHeight uint64) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) insertBlock(block commonBlock) error {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) insertBlocks(blocks []commonBlock) error {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) head() commonBlock {
//	return newMockCommonBlock(1, 1)
//}
//
//func (*mockSnapshotS) getBlock(height uint64) commonBlock {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) InsertAccountBlocks(vmAccountBlocks []*vm_db.VmAccountBlock) error {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) GetLatestAccountBlock(addr *types.Address) (*ledger.AccountBlock, error) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) GetAccountBlockByHeight(addr *types.Address, height uint64) (*ledger.AccountBlock, error) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) DeleteAccountBlocks(addr *types.Address, toHeight uint64) (map[types.Address][]*ledger.AccountBlock, error) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) GetUnConfirmAccountBlocks(addr *types.Address) []*ledger.AccountBlock {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) GetFirstConfirmedAccountBlockBySbHeight(snapshotBlockHeight uint64, addr *types.Address) (*ledger.AccountBlock, error) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) error {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) DeleteSnapshotBlocksToHeight(toHeight uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) BroadcastSnapshotBlock(block *ledger.SnapshotBlock) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) BroadcastSnapshotBlocks(blocks []*ledger.SnapshotBlock) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) BroadcastAccountBlock(block *ledger.AccountBlock) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) BroadcastAccountBlocks(blocks []*ledger.AccountBlock) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) FetchSnapshotBlocks(start types.Hash, count uint64) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) FetchAccountBlocks(start types.Hash, count uint64, address *types.Address) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) SubscribeAccountBlock(fn net.AccountblockCallback) (subId int) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) UnsubscribeAccountBlock(subId int) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) SubscribeSnapshotBlock(fn net.SnapshotBlockCallback) (subId int) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) UnsubscribeSnapshotBlock(subId int) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) SubscribeSyncStatus(fn net.SyncStateCallback) (subId int) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) UnsubscribeSyncStatus(subId int) {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) SyncState() net.SyncState {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) VerifyNetSb(block *ledger.SnapshotBlock) error {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) VerifyReferred(block *ledger.SnapshotBlock) *verifier.SnapshotBlockVerifyStat {
//	panic("implement me")
//}
//
//func (*mockSnapshotS) VerifyTimeout(nowHeight uint64, referHeight uint64) bool {
//	panic("implement me")
//}

func TestNewSnapshotPool(t *testing.T) {
	v := &ForkVersion{}
	mock := &mockSnapshotS{}
	l := log15.New("module", "mock")
	blacklist, _ := NewBlacklist()
	p := newSnapshotPool("snapshot", v, &snapshotVerifier{v: mock}, &snapshotSyncer{fetcher: mock, log: l}, &snapshotCh{bc: mock, version: v}, blacklist, l)
	po := &pool{}
	p.init(&tools{rw: mock}, po)
	p.Start()
	time.Sleep(8 * time.Second)
}

func TestSelect(t *testing.T) {
	nextCompactTime := time.Now()
	nextInsertTime := time.Now()
	for {
		select {
		default:
			now := time.Now()
			if now.After(nextCompactTime) {
				nextCompactTime = now.Add(50 * time.Millisecond)
				println("nextCompactTime", now.String())
			}

			if now.After(nextInsertTime) {
				nextInsertTime = now.Add(200 * time.Millisecond)
				println("insertTime", now.String())
			}
			n2 := time.Now()
			s1 := nextCompactTime.Sub(n2)
			s2 := nextInsertTime.Sub(n2)
			if s1 > s2 {
				time.Sleep(s2)
			} else {
				time.Sleep(s1)
			}
		}
	}
}

func TestName(t *testing.T) {
	println(time.Millisecond.Nanoseconds())
}
