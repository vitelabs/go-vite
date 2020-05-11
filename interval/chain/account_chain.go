package chain

import (
	"errors"

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
	address       common.Address
	head          *common.AccountStateBlock
	store         store.BlockStore
	listener      face.ChainListener
	snapshotPoint *stack.Stack
	unconfirmed   []*common.AccountStateBlock
}

func newAccountChain(address common.Address, listener face.ChainListener, store store.BlockStore) *accountChain {
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

func (acctCh *accountChain) GetBlockByHeight(height common.Height) *common.AccountStateBlock {
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
func (acctCh *accountChain) GetBlockByHash(address common.Address, hash common.Hash) *common.AccountStateBlock {
	block := acctCh.store.GetAccountByHash(address, hash)
	return block
}

func (acctCh *accountChain) insertHeader(block *common.AccountStateBlock) error {
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
func (acctCh *accountChain) removeHeader(block *common.AccountStateBlock) error {
	log.Info("remove from account Block: %v", block)
	{ // check
		has := acctCh.hasSnapshotPoint(block.Height(), block.Hash())
		if has {
			return errors.New("has snapshot")
		}
		if acctCh.head.Hash() != block.Hash() {
			return errors.New("remove header fail")
		}
	}

	{ // delete header
		acctCh.store.DeleteAccount(acctCh.address, common.HashHeight{Hash: block.Hash(), Height: block.Height()})
		if block.BlockType == common.RECEIVED {
			acctCh.store.DeleteSourceHash(block.Source.Hash)
		}
	}
	unconfirmedLen := len(acctCh.unconfirmed)
	if unconfirmedLen > 0 { // delete unconfirmed
		if acctCh.unconfirmed[unconfirmedLen-1].Hash() != block.Hash() {
			panic("unconfirmed check fail")
		}
		acctCh.unconfirmed = acctCh.unconfirmed[:unconfirmedLen-1]
	}

	prev := acctCh.store.GetAccountByHash(acctCh.address, block.PrevHash())
	acctCh.head = prev
	if prev == nil {
		acctCh.store.SetAccountHead(acctCh.address, nil)
	} else {
		acctCh.store.SetAccountHead(acctCh.address, &common.HashHeight{Hash: prev.Hash(), Height: prev.Height()})
	}
	acctCh.listener.AccountRemoveCallback(acctCh.address, block)
	return nil
}

func (acctCh *accountChain) removeUntil(height common.Height) (result []*common.AccountStateBlock, err error) {
	for {
		head := acctCh.Head()
		if head.Height() > height {
			err = acctCh.removeHeader(head)
			if err != nil {
				return
			}
			result = append(result, head)
		} else {
			break
		}
	}
	return
}
func (acctCh *accountChain) removeTo(height common.Height) (result []*common.AccountStateBlock, err error) {
	for {
		head := acctCh.Head()
		if head.Height() >= height {
			err = acctCh.removeHeader(head)
			if err != nil {
				return
			}
			result = append(result, head)
		} else {
			break
		}
	}
	return
}

func (acctCh *accountChain) getByFromHash(fromHash common.Hash) *common.AccountStateBlock {
	if acctCh.head == nil {
		return nil
	}
	height := acctCh.head.Height()
	for i := height; i > 0; i-- {
		// first block(i==0) is create block
		v := acctCh.store.GetAccountByHeight(acctCh.address, i)
		if v.BlockType == common.RECEIVED && v.Source.Hash == fromHash {
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

func (acctCh *accountChain) SnapshotPoint(snapshotHeight common.Height, snapshotHash common.Hash, h *common.AccountHashH) error {
	// check valid
	head := acctCh.head
	if head == nil {
		return errors.New("account[" + acctCh.address.String() + "] not exist.")
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
		errMsg := "account[" + acctCh.address.String() + "] state error. accHeight: " + h.Height.String() +
			"accHash:" + h.Hash.String() +
			" expAccHeight:" + point.Height().String() +
			" expAccHash:" + point.Hash().String()
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

func (acctCh *accountChain) RollbackTo(height common.Height) ([]*common.AccountStateBlock, error) {
	return acctCh.removeTo(height)
}

func (acctCh *accountChain) RollbackUnconfirmed() ([]*common.AccountStateBlock, error) {
	result := acctCh.unconfirmed

	for _, b := range result {
		acctCh.removeHeader(b)
	}
	return result, nil
}

//SnapshotPoint
func (acctCh *accountChain) RollbackSnapshotPointTo(to *common.SnapshotPoint) (result []*common.AccountStateBlock, err error) {
	{
		point := acctCh.peek()
		if point == nil {
			return
		}
		if point.SnapshotHeight < to.SnapshotHeight {
			return
		}
	}
	for {
		point := acctCh.peek()
		if point == nil {
			break
		}
		if point.SnapshotHeight <= to.SnapshotHeight {
			acctCh.snapshotPoint.Pop()
			continue
		} else {
			break
		}
	}
	{
		util := common.EmptyHashHeight
		point := acctCh.peek()
		if point != nil {
			util = common.HashHeight{
				Hash:   point.AccountHash,
				Height: point.AccountHeight,
			}
		}
		return acctCh.removeUntil(util.Height)
	}
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
func (acctCh *accountChain) hasSnapshotPoint(accountHeight common.Height, accountHash common.Hash) bool {
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
