package ledger

import (
	"sync"

	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/monitor"
)

// just only unreceived transactions

type Req struct {
	ReqHash common.Hash
	acc     *common.AccountHashH
	state   int // 0:dirty  1:confirmed  2:unconfirmed
	Amount  common.Balance
	From    common.Address
}

type reqPool struct {
	accounts map[common.Address]*reqAccountPool
	rw       sync.RWMutex
}

func (rp *reqPool) SnapshotInsertCallback(block *common.SnapshotBlock) {

}

func (rp *reqPool) SnapshotRemoveCallback(block *common.SnapshotBlock) {

}

func (rp *reqPool) AccountInsertCallback(address string, block *common.AccountStateBlock) {
	monitor.LogEvent("chain", "insert")
	rp.blockInsert(block)
}

func (rp *reqPool) AccountRemoveCallback(address string, block *common.AccountStateBlock) {
	rp.blockRollback(block)
}

type reqAccountPool struct {
	reqs map[common.Hash]*Req
}

func newReqPool() *reqPool {
	pool := &reqPool{}
	pool.accounts = make(map[common.Address]*reqAccountPool)
	return pool
}

func (rp *reqPool) blockInsert(block *common.AccountStateBlock) {
	rp.rw.Lock()
	defer rp.rw.Unlock()
	if block.BlockType == common.SEND {
		req := &Req{ReqHash: block.Hash(), state: 2, From: block.From, Amount: block.ModifiedAmount}
		account := rp.account(block.To)
		if account == nil {
			account = &reqAccountPool{reqs: make(map[common.Hash]*Req)}
			rp.accounts[block.To] = account
		}
		account.reqs[req.ReqHash] = req
	} else if block.BlockType == common.RECEIVED {
		//delete(rp.account(block.To).reqs, block.SourceHash)
		req := rp.getReq(block.To, block.Source.Hash)
		req.state = 1
		req.acc = common.NewAccountHashH(block.To, block.Hash(), block.Height())
	}
}

func (rp *reqPool) blockRollback(block *common.AccountStateBlock) {
	rp.rw.Lock()
	defer rp.rw.Unlock()
	if block.BlockType == common.SEND {
		//delete(rp.account(block.To).reqs, block.Hash())
		rp.getReq(block.To, block.Hash()).state = 0
	} else if block.BlockType == common.RECEIVED {
		//req := &Req{reqHash: block.SourceHash}
		//rp.account(block.To).reqs[req.reqHash] = req
		rp.getReq(block.To, block.Source.Hash).state = 2
	}
}

func (rp *reqPool) account(address common.Address) *reqAccountPool {
	pool := rp.accounts[address]
	return pool
}

func (rp *reqPool) getReqs(address common.Address) []*Req {
	rp.rw.RLock()
	defer rp.rw.RUnlock()
	account := rp.account(address)
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
func (rp *reqPool) getReq(address common.Address, sourceHash common.Hash) *Req {
	account := rp.account(address)
	if account == nil {
		return nil
	}
	return account.reqs[sourceHash]
}

func (rp *reqPool) confirmed(address common.Address, sourceHash common.Hash) *common.AccountHashH {
	account := rp.account(address)
	if account == nil {
		return nil
	}
	req := account.reqs[sourceHash]

	if req != nil && req.state == 1 {
		return req.acc
	}
	return nil
}
