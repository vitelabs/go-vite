package pool

import (
	"strconv"

	"github.com/pkg/errors"
	ch "github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	//"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/ledger"
)

type chainRw interface {
	insertBlock(block commonBlock) error
	removeBlock(block commonBlock) error

	insertBlocks(blocks []commonBlock) error

	head() commonBlock
	getBlock(height uint64) commonBlock
}

type accountType uint64

const (
	NONE     accountType = 1
	NORMAL   accountType = 2
	CONTRACT accountType = 3
)

type accountCh struct {
	address types.Address
	rw      ch.Chain
	version *ForkVersion
}

func (self *accountCh) insertBlock(block commonBlock) error {
	if !block.checkForkVersion() {
		return errors.New("error fork version. current:" + self.version.String() + ", target:" + strconv.FormatInt(int64(block.forkVersion()), 10))
	}
	//self.rw.InsertAccountBlocks()
	return nil
	//return self.bc.InsertAccountBlock(self.address, block.(*common.AccountStateBlock))
}

func (self *accountCh) removeBlock(block commonBlock) error {
	return nil
	//return self.bc.RemoveAccountHead(self.address, block.(*common.AccountStateBlock))
}

func (self *accountCh) head() commonBlock {
	//block, _ := self.bc.HeadAccount(self.address)
	//if block == nil {
	//	return nil
	//}
	//return block
	return nil
}

func (self *accountCh) getBlock(height uint64) commonBlock {
	//if height == common.EmptyHeight {
	//	return common.NewAccountBlock(height, "", "", "", time.Unix(0, 0), 0, 0, 0, "", common.SEND, "", "", nil)
	//}
	//block := self.bc.GetAccountByHeight(self.address, height)
	//if block == nil {
	//	return nil
	//}
	//return block
	return nil
}

func (self *accountCh) insertBlocks(sendBlocks []commonBlock) error {
	//if !block.checkForkVersion() {
	//	return errors.New("error fork version. current:" + self.version.String() + ", target:" + strconv.FormatInt(int64(block.forkVersion()), 10))
	//}
	return nil
	//return self.bc.InsertAccountBlock(self.address, block.(*common.AccountStateBlock))
}

func (self *accountCh) getHashByHeight(height uint64) *types.Hash {
	return nil
}

func (self *accountCh) delToHeight(height uint64) ([]commonBlock, map[types.Address][]commonBlock, error) {
	return nil, nil, nil
}

func (self *accountCh) getUnConfirmedBlocks() []*ledger.AccountBlock {
	return nil
}
func (self *accountCh) getFirstUnconfirmedBlock() *ledger.AccountBlock {
	return nil
}

//func (self *accountCh) findAboveSnapshotHeight(height uint64) *common.AccountStateBlock {
//	return self.bc.FindAccountAboveSnapshotHeight(self.address, height)
//}

type snapshotCh struct {
	bc      ch.Chain
	version *ForkVersion
}

func (self *snapshotCh) getBlock(height uint64) commonBlock {
	//head := self.bc.GetSnapshotByHeight(height)
	//if head == nil {
	//	return nil
	//}
	//return head
	return nil
}

func (self *snapshotCh) head() commonBlock {
	//block, _ := self.bc.HeadSnapshot()
	//if block == nil {
	//	return nil
	//}
	//return block
	return nil
}

func (self *snapshotCh) headSnapshot() *ledger.SnapshotBlock {
	//block, _ := self.bc.HeadSnapshot()
	//if block == nil {
	//	return nil
	//}
	//return block
	return nil
}

func (self *snapshotCh) getSnapshotBlockByHash(hash types.Hash) *ledger.SnapshotBlock {
	//head := self.bc.GetSnapshotByHeight(height)
	//if head == nil {
	//	return nil
	//}
	//return head
	return nil
}

func (self *snapshotCh) delToHeight(height uint64) ([]commonBlock, map[types.Address][]commonBlock, error) {
	return nil, nil, nil
}

func (self *snapshotCh) insertBlock(block commonBlock) error {
	//return self.bc.InsertSnapshotBlock(block.(*common.SnapshotBlock))
	return nil
}

func (self *snapshotCh) insertBlocks(sendBlocks []commonBlock) error {
	//if !block.checkForkVersion() {
	//	return errors.New("error fork version. current:" + self.version.String() + ", target:" + strconv.FormatInt(int64(block.forkVersion()), 10))
	//}
	return nil
	//return self.bc.InsertAccountBlock(self.address, block.(*common.AccountStateBlock))
}

func (self *snapshotCh) removeBlock(block commonBlock) error {
	//return self.bc.RemoveSnapshotHead(block.(*common.SnapshotBlock))
	return nil
}
