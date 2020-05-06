package ledger

import (
	"sync"

	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/monitor"
)

// just only unreceived transactions

type Req struct {
	ReqHash string
	acc     *common.AccountHashH
	state   int // 0:dirty  1:confirmed  2:unconfirmed
	Amount  int
	From    string
}

type reqPool struct {
	accounts map[string]*reqAccountPool
	rw       sync.RWMutex
}

func (self *reqPool) SnapshotInsertCallback(block *common.SnapshotBlock) {

}

func (self *reqPool) SnapshotRemoveCallback(block *common.SnapshotBlock) {

}

func (self *reqPool) AccountInsertCallback(address string, block *common.AccountStateBlock) {
	monitor.LogEvent("chain", "insert")
	self.blockInsert(block)
}

func (self *reqPool) AccountRemoveCallback(address string, block *common.AccountStateBlock) {
	self.blockRollback(block)
}

type reqAccountPool struct {
	reqs map[string]*Req
}

func newReqPool() *reqPool {
	pool := &reqPool{}
	pool.accounts = make(map[string]*reqAccountPool)
	return pool
}

func (self *reqPool) blockInsert(block *common.AccountStateBlock) {
	self.rw.Lock()
	defer self.rw.Unlock()
	if block.BlockType == common.SEND {
		req := &Req{ReqHash: block.Hash(), state: 2, From: block.From, Amount: block.ModifiedAmount}
		account := self.account(block.To)
		if account == nil {
			account = &reqAccountPool{reqs: make(map[string]*Req)}
			self.accounts[block.To] = account
		}
		account.reqs[req.ReqHash] = req
	} else if block.BlockType == common.RECEIVED {
		//delete(self.account(block.To).reqs, block.SourceHash)
		req := self.getReq(block.To, block.Source.Hash)
		req.state = 1
		req.acc = common.NewAccountHashH(block.To, block.Hash(), block.Height())
	}
}

func (self *reqPool) blockRollback(block *common.AccountStateBlock) {
	self.rw.Lock()
	defer self.rw.Unlock()
	if block.BlockType == common.SEND {
		//delete(self.account(block.To).reqs, block.Hash())
		self.getReq(block.To, block.Hash()).state = 0
	} else if block.BlockType == common.RECEIVED {
		//req := &Req{reqHash: block.SourceHash}
		//self.account(block.To).reqs[req.reqHash] = req
		self.getReq(block.To, block.Source.Hash).state = 2
	}
}

func (self *reqPool) account(address string) *reqAccountPool {
	pool := self.accounts[address]
	return pool
}

func (self *reqPool) getReqs(address string) []*Req {
	self.rw.RLock()
	defer self.rw.RUnlock()
	account := self.account(address)
	var result []*Req
	if account == nil {
		return result
	}
	for _, req := range account.reqs {
		if req.state != 1 {
			result = append(result, req)
		}
	}
	return result
}
func (self *reqPool) getReq(address string, sourceHash string) *Req {
	account := self.account(address)
	if account == nil {
		return nil
	}
	return account.reqs[sourceHash]
}

func (self *reqPool) confirmed(address string, sourceHash string) *common.AccountHashH {
	account := self.account(address)
	if account == nil {
		return nil
	}
	req := account.reqs[sourceHash]

	if req != nil && req.state == 1 {
		return req.acc
	}
	return nil
}
