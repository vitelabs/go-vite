package pool

import (
	"strconv"

	"time"

	"github.com/pkg/errors"
	ch "github.com/vitelabs/go-vite/interval/chain"
	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/version"
)

type chainRw interface {
	insertChain(block common.Block, forkVersion int) error
	removeChain(block common.Block) error

	head() common.Block
	getBlock(height uint64) common.Block
}

type accountCh struct {
	address string
	bc      ch.BlockChain
	version *version.Version
}

func (acctCh *accountCh) insertChain(block common.Block, forkVersion int) error {
	if forkVersion != acctCh.version.Val() {
		return errors.New("error fork version. current:" + acctCh.version.String() + ", target:" + strconv.Itoa(forkVersion))
	}
	return acctCh.bc.InsertAccountBlock(acctCh.address, block.(*common.AccountStateBlock))
}

func (acctCh *accountCh) removeChain(block common.Block) error {
	return acctCh.bc.RemoveAccountHead(acctCh.address, block.(*common.AccountStateBlock))
}

func (acctCh *accountCh) head() common.Block {
	block, _ := acctCh.bc.HeadAccount(acctCh.address)
	if block == nil {
		return nil
	}
	return block
}

func (acctCh *accountCh) getBlock(height uint64) common.Block {
	if height == common.EmptyHeight {
		return common.NewAccountBlock(height, "", "", "", time.Unix(0, 0), 0, 0, common.SEND, "", "", nil)
	}
	block := acctCh.bc.GetAccountByHeight(acctCh.address, height)
	if block == nil {
		return nil
	}
	return block
}
func (acctCh *accountCh) findAboveSnapshotHeight(height uint64) *common.AccountStateBlock {
	return acctCh.bc.FindAccountAboveSnapshotHeight(acctCh.address, height)
}

type snapshotCh struct {
	bc      ch.BlockChain
	version *version.Version
}

func (self *snapshotCh) getBlock(height uint64) common.Block {
	head := self.bc.GetSnapshotByHeight(height)
	if head == nil {
		return nil
	}
	return head
}

func (self *snapshotCh) head() common.Block {
	block, _ := self.bc.HeadSnapshot()
	if block == nil {
		return nil
	}
	return block
}

func (self *snapshotCh) insertChain(block common.Block, forkVersion int) error {
	return self.bc.InsertSnapshotBlock(block.(*common.SnapshotBlock))
}

func (self *snapshotCh) removeChain(block common.Block) error {
	return self.bc.RemoveSnapshotHead(block.(*common.SnapshotBlock))
}
