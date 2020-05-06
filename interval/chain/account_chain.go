package chain

import (
	"errors"
	"strconv"

	"fmt"

	"time"

	"github.com/golang-collections/collections/stack"
	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/face"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/monitor"
	"github.com/vitelabs/go-vite/interval/store"
)

// account block chain
type accountChain struct {
	address       string
	head          *common.AccountStateBlock
	store         store.BlockStore
	listener      face.ChainListener
	snapshotPoint *stack.Stack
}

func newAccountChain(address string, listener face.ChainListener, store store.BlockStore) *accountChain {
	self := &accountChain{}
	self.address = address
	self.store = store
	head := self.store.GetAccountHead(self.address)
	if head != nil {
		self.head = self.store.GetAccountByHeight(self.address, head.Height)
	}

	self.listener = listener
	self.snapshotPoint = stack.New()
	return self
}

func (acctCh *accountChain) Head() *common.AccountStateBlock {
	return acctCh.head
}

func (acctCh *accountChain) GetBlockByHeight(height uint64) *common.AccountStateBlock {
	if height < common.FirstHeight {
		log.Error("can't request height 0 block. account:%s", acctCh.address)
		return nil
	}
	block := acctCh.store.GetAccountByHeight(acctCh.address, height)

	return block
}

func (acctCh *accountChain) GetBlockByHashH(hashH common.HashHeight) *common.AccountStateBlock {
	if hashH.Height < common.FirstHeight {
		log.Error("can't request height 0 block. account:%s", acctCh.address)
		return nil
	}
	block := acctCh.store.GetAccountByHeight(acctCh.address, hashH.Height)
	if block != nil && block.Hash() == hashH.Hash {
		return block
	}
	return nil
}
func (acctCh *accountChain) GetBlockByHash(address string, hash string) *common.AccountStateBlock {
	block := acctCh.store.GetAccountByHash(address, hash)
	return block
}

func (acctCh *accountChain) insertChain(block *common.AccountStateBlock) error {
	defer monitor.LogTime("chain", "accountInsert", time.Now())
	log.Info("insert to account Chain: %v", block)
	acctCh.store.PutAccount(acctCh.address, block)
	acctCh.head = block
	acctCh.listener.AccountInsertCallback(acctCh.address, block)
	acctCh.store.SetAccountHead(acctCh.address, &common.HashHeight{Hash: block.Hash(), Height: block.Height()})

	if block.BlockType == common.RECEIVED {
		acctCh.store.PutSourceHash(block.Source.Hash, common.NewAccountHashH(acctCh.address, block.Hash(), block.Height()))
	}
	return nil
}
func (acctCh *accountChain) removeChain(block *common.AccountStateBlock) error {
	log.Info("remove from account Chain: %v", block)
	has := acctCh.hasSnapshotPoint(block.Height(), block.Hash())
	if has {
		return errors.New("has snapshot.")
	}

	head := acctCh.store.GetAccountByHash(acctCh.address, block.Hash())
	acctCh.store.DeleteAccount(acctCh.address, common.HashHeight{Hash: block.Hash(), Height: block.Height()})
	acctCh.listener.AccountRemoveCallback(acctCh.address, block)
	acctCh.head = head
	if head == nil {
		acctCh.store.SetAccountHead(acctCh.address, nil)
	} else {
		acctCh.store.SetAccountHead(acctCh.address, &common.HashHeight{Hash: head.Hash(), Height: head.Height()})
	}
	if block.BlockType == common.RECEIVED {
		acctCh.store.DeleteSourceHash(block.Source.Hash)
	}
	return nil
}

func (acctCh *accountChain) findAccountAboveSnapshotHeight(snapshotHeight uint64) *common.AccountStateBlock {
	if acctCh.head == nil {
		return nil
	}
	for i := acctCh.head.Height(); i >= common.FirstHeight; i-- {
		block := acctCh.store.GetAccountByHeight(acctCh.address, i)
		if block.SnapshotHeight <= snapshotHeight {
			return block
		}
	}
	return nil
}
func (acctCh *accountChain) getBySourceBlock(sourceHash string) *common.AccountStateBlock {
	if acctCh.head == nil {
		return nil
	}
	height := acctCh.head.Height()
	for i := height; i > 0; i-- {
		// first block(i==0) is create block
		v := acctCh.store.GetAccountByHeight(acctCh.address, i)
		if v.BlockType == common.RECEIVED && v.Source.Hash == sourceHash {
			return v
		}
	}
	return nil
}

func (acctCh *accountChain) NextSnapshotPoint() (*common.AccountHashH, error) {
	var lastPoint *common.SnapshotPoint
	p := acctCh.snapshotPoint.Peek()
	if p != nil {
		lastPoint = p.(*common.SnapshotPoint)
	}

	if lastPoint == nil {
		if acctCh.head != nil {
			return common.NewAccountHashH(acctCh.address, acctCh.head.Hash(), acctCh.head.Height()), nil
		}
	} else {
		if lastPoint.AccountHeight < acctCh.head.Height() {
			return common.NewAccountHashH(acctCh.address, acctCh.head.Hash(), acctCh.head.Height()), nil
		}
	}
	return nil, errors.New("not found")
}

func (acctCh *accountChain) SnapshotPoint(snapshotHeight uint64, snapshotHash string, h *common.AccountHashH) error {
	// check valid
	head := acctCh.head
	if head == nil {
		return errors.New("account[" + acctCh.address + "] not exist.")
	}

	var lastPoint *common.SnapshotPoint
	p := acctCh.snapshotPoint.Peek()
	if p != nil {
		lastPoint = p.(*common.SnapshotPoint)
		if snapshotHeight <= lastPoint.SnapshotHeight ||
			h.Height < lastPoint.AccountHeight {
			errMsg := fmt.Sprintf("acount snapshot point check fail.sHeight:[%d], lastSHeight:[%d], aHeight:[%d], lastAHeight:[%d]",
				snapshotHeight, lastPoint.SnapshotHeight, h.Height, lastPoint.AccountHeight)
			return errors.New(errMsg)
		}
	}

	point := acctCh.GetBlockByHeight(h.Height)
	if h.Hash == point.Hash() && h.Height == point.Height() {
		point := &common.SnapshotPoint{SnapshotHeight: snapshotHeight, SnapshotHash: snapshotHash, AccountHash: h.Hash, AccountHeight: h.Height}
		acctCh.snapshotPoint.Push(point)
		return nil
	} else {
		errMsg := "account[" + acctCh.address + "] state error. accHeight: " + strconv.FormatUint(h.Height, 10) +
			"accHash:" + h.Hash +
			" expAccHeight:" + strconv.FormatUint(point.Height(), 10) +
			" expAccHash:" + point.Hash()
		return errors.New(errMsg)
	}
}

//SnapshotPoint
func (acctCh *accountChain) RollbackSnapshotPoint(start *common.SnapshotPoint, end *common.SnapshotPoint) error {
	point := acctCh.peek()
	if point == nil {
		return errors.New("not exist snapshot point")
	}
	if !point.Equals(start) {
		return errors.New("not equals for start")
	}
	for {
		point := acctCh.peek()
		if point == nil {
			return errors.New("not exist snapshot point")
		}
		if point.AccountHeight <= end.AccountHeight {
			acctCh.snapshotPoint.Pop()
		} else {
			break
		}
		if point.AccountHeight == end.AccountHeight {
			break
		}
	}
	return nil
}

//func (self *accountChain) rollbackSnapshotPoint(start *common.SnapshotPoint) error {
//	point := self.peek()
//	if point == nil {
//		return errors.New("not exist snapshot point."}
//	}
//
//	if point.SnapshotHash == start.SnapshotHash &&
//		point.SnapshotHeight == start.SnapshotHeight &&
//		point.AccountHeight == start.AccountHeight &&
//		point.AccountHash == start.AccountHash {
//		self.snapshotPoint.Pop()
//		return nil
//	}
//
//	errMsg := "account[" + self.address + "] state error. expect:" + point.String() +
//		", actual:" + start.String()
//	return errors.New( errMsg}
//}

//SnapshotPoint ddd
func (acctCh *accountChain) hasSnapshotPoint(accountHeight uint64, accountHash string) bool {
	point := acctCh.peek()
	if point == nil {
		return false
	}

	if point.AccountHeight >= accountHeight {
		return true
	}
	return false

}

func (acctCh *accountChain) peek() *common.SnapshotPoint {
	var lastPoint *common.SnapshotPoint
	p := acctCh.snapshotPoint.Peek()
	if p != nil {
		lastPoint = p.(*common.SnapshotPoint)
	}
	return lastPoint
}
