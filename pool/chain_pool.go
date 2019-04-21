package pool

import (
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/vitelabs/go-vite/pool/tree"

	"sync"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
)

type chainPool struct {
	poolId          string
	log             log15.Logger
	lastestChainIdx int32
	//current         *forkedChain
	snippetChains map[string]*snippetChain // head is fixed

	tree tree.Tree
	//chains          map[string]*forkedChain
	diskChain *branchChain

	chainMu sync.Mutex
}

func (self *chainPool) forkChain(forked tree.Branch, snippet *snippetChain) (tree.Branch, error) {
	//new := &forkedChain{}
	//
	//new.heightBlocks = snippet.heightBlocks
	//new.tailHeight = snippet.tailHeight
	//new.tailHash = snippet.tailHash
	//new.headHeight = snippet.headHeight
	//new.headHash = snippet.headHash
	//new.referChain = forked
	//
	//new.chainId = self.genChainId()
	//
	//self.addChain(new)

	new := self.tree.ForkBranch(forked, snippet.tailHeight, snippet.tailHash)
	for _, v := range snippet.heightBlocks {
		new.AddHead(v)
	}

	return new, nil
}

func (self *chainPool) forkFrom(forked tree.Branch, height uint64, hash types.Hash) (tree.Branch, error) {
	forkedHeight, forkedHash := forked.HeadHH()
	if forkedHash == hash && forkedHeight == height {
		return forked, nil
	}

	new := self.tree.ForkBranch(forked, height, hash)
	return new, nil
}

func (self *chainPool) genChainId() string {
	return self.poolId + "-" + strconv.Itoa(self.incChainIdx())
}

func (self *chainPool) incChainIdx() int {
	for {
		old := self.lastestChainIdx
		new := old + 1
		if atomic.CompareAndSwapInt32(&self.lastestChainIdx, old, new) {
			return int(new)
		} else {
			self.log.Info(fmt.Sprintf("lastest forkedChain idx concurrent for %d.", old))
		}
	}
}
func (self *chainPool) init() {
	//initBlock := self.diskChain.Head()
	//self.current.init(initBlock)
	//self.current.referChain = self.diskChain
	//self.chains = make(map[string]*forkedChain)
	self.snippetChains = make(map[string]*snippetChain)
	self.tree.Init(self.poolId, self.diskChain)
	//self.addChain(self.current)
}

/** constrained condition:
disk header block can be found in chain.
*/
func (self *chainPool) currentModifyToChain(chain tree.Branch) error {
	return self.tree.SwitchMainTo(chain)
	//if chain.id() == self.current.id() {
	//	return nil
	//}
	//chain.prune()
	//
	//err := self.checkAncestor(chain, self.diskChain)
	//if err != nil {
	//	return err
	//}
	//
	//e := self.check()
	//if e != nil {
	//	self.log.Error("---------[1]", "err", e)
	//}
	//
	//for chain.referChain.id() != self.diskChain.id() {
	//	fromChain := chain.referChain.(*forkedChain)
	//
	//	err := self.exchangeRefer(fromChain, chain)
	//	if err != nil {
	//		return err
	//	}
	//}
	//self.log.Warn("current modify.", "from", self.current.id(), "to", chain.id(),
	//	"fromTailHeight", self.current.tailHeight, "fromHeadHeight", self.current.headHeight,
	//	"toTailHeight", chain.tailHeight, "toHeadHeight", chain.headHeight)
	//self.current = chain
	//e = self.check()
	//if e != nil {
	//	self.log.Error("---------[2]", "err", e)
	//}
	////self.modifyChainRefer()
	//return nil
}

//func (self *chainPool) modifyRefer(from *forkedChain, to *forkedChain) error {
//	r := reduceChainByRefer(to)
//	if len(r) > 0 {
//		self.log.Debug("modifyRefer-clear ChainBase", "chainId", to.id(), "start", r[0].Height(), "end", r[len(r)-1].Height())
//	}
//	// from.tailHeight <= to.tailHeight  && from.headHeight > to.tail.Height
//	toTailHeight := to.tailHeight
//	fromTailHeight := from.tailHeight
//	fromHeadHeight := from.headHeight
//	if fromTailHeight <= toTailHeight && fromHeadHeight >= toTailHeight {
//		for i := toTailHeight; i > fromTailHeight; i-- {
//			w := from.getBlock(i, false)
//			if w != nil {
//				to.addTail(w)
//			}
//		}
//		for i := fromTailHeight + 1; i <= toTailHeight; i++ {
//			w := from.getBlock(i, false)
//			if w != nil {
//				from.removeTail(w)
//			}
//		}
//
//		to.referChain = from.referChain
//		from.referChain = to
//
//		e := self.modifyReferForChainExchange(from, to, to)
//
//		self.log.Info("modify refer", "from", from.id(), "to", to.id(),
//			"fromTailHeight", fromTailHeight, "fromHeadHeight", fromHeadHeight,
//			"toTailHeight", toTailHeight, "toHeadHeight", to.headHeight, "err", e)
//		return e
//	} else {
//		return errors.Errorf("err for modifyRefer.from:%s, to:%s, fromTailHeight:%d, fromHeadHeight:%d, toTailHeight:%d, toHeadHeight:%d",
//			from.id(), to.id(), fromTailHeight, fromHeadHeight, toTailHeight, to.headHeight)
//
//	}
//}
//func (self *chainPool) modifyReferForChainExchange(from *forkedChain, to *forkedChain, diskInstead *forkedChain) error {
//	toTailHeight := to.tailHeight
//	fromTailHeight := from.tailHeight
//	fromHeadHeight := from.headHeight
//	for _, v := range self.allChain() {
//		if v.id() == from.id() || v.id() == to.id() {
//			continue
//		}
//		if v.referChain.id() == from.id() {
//			block, reader := to.getBlockByChain(v.tailHeight)
//			if block != nil && block.Hash() == v.tailHash {
//				if reader.id() == self.diskChain.id() {
//					self.log.Info("modify refer[7]", "from", from.id(), "to", to.id(),
//						"fromTailHeight", fromTailHeight, "fromHeadHeight", fromHeadHeight,
//						"toTailHeight", toTailHeight, "toHeadHeight", to.headHeight,
//						"v", v.id(),
//						"vTailHeight", v.tailHeight, "vTailHash", v.tailHash)
//					v.referChain = diskInstead
//				} else {
//					self.log.Info("modify refer[8]", "from", from.id(), "to", to.id(),
//						"fromTailHeight", fromTailHeight, "fromHeadHeight", fromHeadHeight,
//						"toTailHeight", toTailHeight, "toHeadHeight", to.headHeight,
//						"v", v.id(),
//						"vTailHeight", v.tailHeight, "vTailHash", v.tailHash,
//						"readerId", reader.id())
//					v.referChain = reader
//				}
//			} else {
//				self.log.Warn("modify refer[4]", "from", from.id(), "to", to.id(),
//					"fromTailHeight", fromTailHeight, "fromHeadHeight", fromHeadHeight,
//					"toTailHeight", toTailHeight, "toHeadHeight", to.headHeight,
//					"v", v.id(),
//					"vTailHeight", v.tailHeight, "vTailHash", v.tailHash)
//			}
//		}
//	}
//	return nil
//}
//
//func (self *chainPool) modifyChainRefer() {
//	cs := self.allChain()
//	for _, c := range cs {
//		if c.id() == self.current.id() {
//			continue
//		}
//		b, reader := c.referChain.getBlockByChain(c.tailHeight)
//		if b != nil {
//			if reader.id() == self.diskChain.id() {
//				if c.referChain.id() == self.current.id() {
//					continue
//				}
//				c.referChain = self.current
//				self.log.Warn("[1]modify for refer.", "from", c.id(), "refer", c.referChain.id(), "tailHeight", c.tailHeight)
//				continue
//			}
//
//			if reader.id() == c.referChain.id() {
//				continue
//			}
//
//			if c.id() == reader.id() {
//				self.log.Error("err for modifyChainRefer refer self.", "from", c.id(), "refer", c.referChain.id(), "tailHeight", c.tailHeight)
//				continue
//			}
//			self.log.Warn("[2]modify for refer.", "from", c.id(), "refer", c.referChain.id(), "tailHeight", c.tailHeight)
//			c.referChain = reader
//		} else {
//			self.log.Error("err for modifyChainRefer.", "from", c.id(), "refer", c.referChain.id(), "tailHeight", c.tailHeight)
//			for _, v := range cs {
//				b2, r2 := v.getBlockByChain(c.tailHeight)
//				if b2 != nil {
//					if r2.id() == self.diskChain.id() {
//						self.log.Warn("[3]modify for refer.", "from", c.id(), "refer", c.referChain.id(), "tailHeight", c.tailHeight)
//						c.referChain = self.current
//						break
//					}
//					if r2.id() == c.id() {
//						self.log.Error("err for modifyChainRefer refer self r2.", "from", c.id(), "refer", c.referChain.id(), "tailHeight", c.tailHeight)
//						continue
//					}
//					self.log.Warn("[4]modify for refer.", "from", c.id(), "refer", c.referChain.id(), "tailHeight", c.tailHeight)
//					c.referChain = r2
//					break
//				}
//			}
//		}
//	}
//}

func (self *chainPool) currentModify(initBlock commonBlock) {
	//new := &forkedChain{}
	//new.chainId = self.genChainId()
	//new.init(initBlock)
	//new.referChain = self.diskChain
	//self.addChain(new)
	//c := self.current
	//c.referChain = new
	//self.current = new
	main := self.tree.Main()
	new := self.tree.ForkBranch(main, initBlock.Height(), initBlock.Hash())
	self.tree.SwitchMainTo(new)
	//self.modifyReferForChainExchange(c, new, new)
}
func (self *chainPool) fork2(snippet *snippetChain, chains map[string]tree.Branch) (bool, bool, tree.Branch, error) {

	var forky, insertable bool
	var result tree.Branch = nil
	var hr tree.Branch

	var err error

	trace := ""

LOOP:
	for _, c := range chains {
		tH := snippet.tailHeight
		tHash := snippet.tailHash
		block, reader := c.GetKnotAndBranch(tH)
		if block == nil || block.Hash() != tHash {
			continue
		}

		for i := tH + 1; i <= snippet.headHeight; i++ {
			trace = ""
			b2, r2 := c.GetKnotAndBranch(i)
			sb := snippet.getBlock(i)
			if b2 == nil {
				forky = false
				insertable = true
				hr = reader
				trace += "[1]"
				break LOOP
			}
			if b2.Hash() != sb.Hash() {
				if r2.Id() == reader.Id() {
					forky = true
					insertable = false
					hr = reader
					trace += "[2]"
					break LOOP
				}

				rHeight, rHash := reader.HeadHH()
				if rHeight == tH && rHash == tHash {
					forky = false
					insertable = true
					hr = reader
					trace += "[3]"
					break LOOP
				}
				break
			} else {
				reader = r2
				block = b2
				// todo
				self.log.Info(fmt.Sprintf("block[%s-%d] exists. del from tail.", sb.Hash(), sb.Height()))
				tail := snippet.remTail()
				if tail == nil {
					delete(self.snippetChains, snippet.id())
					hr = nil
					trace += "[4]"
					err = errors.Errorf("snippet rem nil. size:%d", snippet.size())
					break LOOP
				}
				if snippet.size() == 0 {
					delete(self.snippetChains, snippet.id())
					hr = nil
					trace += "[5]"
					err = errors.New("snippet is empty.")
					break LOOP
				}
				tH = tail.Height()
				tHash = tail.Hash()
			}
		}
	}

	if err != nil {
		return false, false, nil, err
	}

	if hr == nil {
		return false, false, nil, nil
	}
	switch hr.Type() {
	case tree.Disk:
		result = self.tree.Main()
		if insertable {
			mHeight, mHash := self.tree.Main().HeadHH()
			if mHeight == snippet.tailHeight && mHash == snippet.tailHash {
				forky = false
				insertable = true
				result = self.tree.Main()
				trace += "[5]"
			} else {
				forky = true
				insertable = false
				result = self.tree.Main()
				trace += "[6]"
			}
		}
	case tree.Normal:
		trace += "[7]"
		result = hr
	}
	// todo
	//if insertable {
	//	err := result.canAddHead(snippet.getBlock(snippet.tailHeight + 1))
	//	if err != nil {
	//		self.log.Error("fork2 fail.",
	//			"sTailHeight", snippet.tailHeight, "sHeadHeight",
	//			snippet.headHeight, "cTailHeight", result.tailHeight, "cHeadHeight", result.headHeight,
	//			"trace", trace)
	//		return forky, false, result, nil
	//	}
	//}

	return forky, insertable, result, nil
}

func (self *chainPool) insertSnippet(c tree.Branch, snippet *snippetChain) error {
	for i := snippet.tailHeight + 1; i <= snippet.headHeight; i++ {
		w := snippet.heightBlocks[i]
		err := self.insert(c, w)
		if err != nil {
			return err
		}
		snippet.deleteTail(w)
	}
	if snippet.tailHeight == snippet.headHeight {
		delete(self.snippetChains, snippet.chainId)
	}
	return nil
}

type ForkChainError struct {
	What string
}

func (e ForkChainError) Error() string {
	return fmt.Sprintf("%s", e.What)
}
func (self *chainPool) insert(c tree.Branch, wrapper commonBlock) error {
	if self.tree.Main().Id() == c.Id() {
		// todo remove
		self.log.Info(fmt.Sprintf("insert to current[%s]:[%s-%d]%s", c.Id(), wrapper.Hash(), wrapper.Height(), wrapper.Latency()))
	} else {
		self.log.Info(fmt.Sprintf("insert to chain[%s]:[%s-%d]%s", c.Id(), wrapper.Hash(), wrapper.Height(), wrapper.Latency()))
	}
	height, hash := c.HeadHH()
	if wrapper.Height() == height+1 {
		if hash == wrapper.PrevHash() {
			c.AddHead(wrapper)
			return nil
		} else {
			return errors.Errorf("forkedChain fork, fork point height[%d],hash[%s], but next block[%s]'s preHash is [%s]",
				height, hash, wrapper.Hash(), wrapper.PrevHash())
		}
	} else {
		return errors.Errorf("forkedChain fork, fork point height[%d],hash[%s], but next block[%s]'s preHash[%s]-[%d]",
			height, hash, wrapper.Hash(), wrapper.PrevHash(), wrapper.Height())
	}
}

func (self *chainPool) insertNotify(head commonBlock) {
	main := self.tree.Main()
	if main.MatchHead(head.PrevHash()) {
		main.AddHead(head)
	} else {
		branch, err := self.genDirectBlock(head)
		if err != nil {
			panic(err)
		}
		self.currentModifyToChain(branch)
	}
	self.tree.Main().RemoveTail(head)
}

func (self *chainPool) genDirectBlock(head commonBlock) (tree.Branch, error) {
	fchain, err := self.forkFrom(self.tree.Main(), head.Height()-1, head.PrevHash())
	if err != nil {
		return nil, err
	}

	fchain.AddHead(head)
	return fchain, nil
}

//func (self *chainPool) writeToChain(chain *forkedChain, block commonBlock) error {
//	height := block.Height()
//	hash := block.Hash()
//	err := self.diskChain.rw.insertBlock(block)
//	if err == nil {
//		chain.removeTail(block)
//		//self.fixReferInsert(chain, self.diskChain, height)
//		return nil
//	} else {
//		self.log.Error(fmt.Sprintf("waiting pool insert forkedChain fail. height:[%d], hash:[%s]", height, hash), "err", err)
//		return err
//	}
//}

func (self *chainPool) writeBlocksToChain(chain tree.Branch, blocks []commonBlock) error {
	if len(blocks) == 0 {
		return nil
	}
	err := self.diskChain.rw.insertBlocks(blocks)

	if err != nil {
		// todo opt log
		self.log.Error(fmt.Sprintf("pool insert Chain fail. height:[%d], hash:[%s], len:[%d]", blocks[0].Height(), blocks[0].Hash(), len(blocks)))
		return err
	}
	for _, b := range blocks {
		chain.RemoveTail(b)
	}
	return nil
}
func (self *chainPool) check() error {
	diskId := self.diskChain.Id()
	currentId := self.tree.Main().Id()
	for _, c := range self.allChain() {
		// refer to disk
		if c.Root().Id() == diskId {
			if c.Id() != currentId {
				self.log.Error(fmt.Sprintf("chain:%s, refer disk.", c.Id()))
				return errors.New("refer disk")
			} else {
				err := checkHeadTailLink(c, c.Root())
				if err != nil {
					self.log.Error(err.Error())
					return err
				}
			}
		} else if c.Root().Id() == currentId {
			// refer to current
			err := checkLink(c, c.Root(), true)
			if err != nil {
				self.log.Error(err.Error())
				return err
			}
		} else {
			err := checkLink(c, c.Root(), false)
			if err != nil {
				self.log.Error(err.Error())
				return err
			}
		}
	}
	return nil
}

//func (self *chainPool) addChain(c *forkedChain) {
//	self.chainMu.Lock()
//	defer self.chainMu.Unlock()
//	self.chains[c.id()] = c
//}
//func (self *chainPool) getChain(id string) *forkedChain {
//	self.chainMu.Lock()
//	defer self.chainMu.Unlock()
//	return self.chains[id]
//}
func (self *chainPool) allChain() map[string]tree.Branch {
	return self.tree.Branches()
}

//func (self *chainPool) clearRepeatChain() []*forkedChain {
//	self.chainMu.Lock()
//	defer self.chainMu.Unlock()
//
//	var r []*forkedChain
//	for id1, c1 := range self.chains {
//		if id1 == self.current.id() {
//			continue
//		}
//		for _, c2 := range self.chains {
//			if c1.tailHeight == c2.tailHeight &&
//				c1.tailHash == c2.tailHash {
//				h := c2.getHeightBlock(c1.headHeight)
//				if h != nil && h.Hash() == c1.headHash {
//					r = append(r, c1)
//				}
//			}
//		}
//	}
//	return r
//}
//func (self *chainPool) delChain(id string) {
//	self.chainMu.Lock()
//	defer self.chainMu.Unlock()
//	delete(self.chains, id)
//}

func (self *chainPool) size() int {
	return -1
	//self.chainMu.Lock()
	//defer self.chainMu.Unlock()
	//return len(self.chains)
}

//func (self *chainPool) findOtherChainsByTail(cur *forkedChain, hash types.Hash, height uint64) []*forkedChain {
//	self.chainMu.Lock()
//	defer self.chainMu.Unlock()
//	var result []*forkedChain
//	for _, v := range self.chains {
//		if v.id() == cur.id() {
//			continue
//		}
//		if v.tailHeight == height {
//			if v.tailHash == hash {
//				result = append(result, v)
//			}
//			continue
//		}
//		b := v.getHeightBlock(height)
//		if b != nil && b.Hash() == hash && v.tailHeight < height {
//			result = append(result, v)
//			continue
//		}
//	}
//	return result
//}

//func (self *chainPool) findEmptyForHead(head commonBlock) *forkedChain {
//	for _, c := range self.chains {
//		if c.size() == uint64(0) && c.headHash == head.Hash() {
//			return c
//		}
//	}
//	return nil
//}
//func (self *chainPool) checkAncestor(c *forkedChain, ancestor heightChainReader) error {
//	head := ancestor.Head()
//	ancestorBlock := c.getBlock(head.Height(), true)
//	if ancestorBlock == nil {
//		chead := c.Head()
//		err := errors.Errorf("check ancestor fail, ancestorBlock is nil. head:[%s][%d], chead:[%s][%d][%d]", head.Hash(), head.Height(), chead.Hash(), chead.Height(), c.headHeight)
//		// todo
//		panic(err)
//		return err
//	}
//	if ancestorBlock.Hash() != head.Hash() {
//		return errors.Errorf("check ancestor fail, ancestoreHash:[%s], headHash:[%s]", ancestorBlock.Hash(), head.Hash())
//	}
//	return nil
//}

//func (self *chainPool) exchangeRefer(from *forkedChain, to *forkedChain) error {
//	e := self.checkExchangeRefer(from, to)
//	if e != nil {
//		r := reduceChainByRefer(to)
//		if len(r) > 0 {
//			self.log.Debug("currentModifyToChain[2]-clear ChainBase", "chainId", to.id(), "start", r[0].Height(), "end", r[len(r)-1].Height())
//		}
//		err := self.checkExchangeRefer(from, to)
//		if err != nil {
//			return err
//		} else {
//			self.log.Warn("check exchange fail. but recover success.", "err", e)
//		}
//	}
//	if from.size() == 0 {
//		r := reduceChainByRefer(to)
//		if len(r) > 0 {
//			self.log.Info("currentModifyToChain[3]-clear ChainBase", "chainId", to.id(), "start", r[0].Height(), "end", r[len(r)-1].Height())
//		}
//		self.log.Error("modify refer[6]", "from", from.id(), "to", to.id(),
//			"fromTailHeight", from.tailHeight, "fromHeadHeight", from.headHeight,
//			"toTailHeight", to.tailHeight, "toHeadHeight", to.headHeight)
//		to.referChain = from.referChain
//		err := self.modifyReferForChainExchange(from, to, to)
//		if err != nil {
//			return err
//		}
//		self.delChain(from.id())
//		return nil
//	} else {
//		err := self.modifyRefer(from, to)
//		if err != nil {
//			return err
//		}
//	}
//
//	r := reduceChainByRefer(to)
//	if len(r) > 0 {
//		self.log.Debug("currentModifyToChain[2]-clear ChainBase", "chainId", to.id(), "start", r[0].Height(), "end", r[len(r)-1].Height())
//	}
//	return nil
//}
//func (self *chainPool) checkExchangeRefer(from *forkedChain, to *forkedChain) error {
//	if to.tailHash == from.tailHash {
//		return nil
//	}
//
//	tailBlock := from.getBlock(to.tailHeight, false)
//	if tailBlock == nil {
//		return errors.Errorf("[%s] and [%s] can't exchange. height[%d], A-Hash[%s], B-Hash[empty], %s,%s",
//			to.id(), from.id(), to.tailHeight, to.tailHash, from.Details(), to.Details())
//	}
//	if tailBlock.Hash() == to.tailHash {
//		return nil
//	} else {
//		return errors.Errorf("[%s] and [%s] can't exchange. height[%d], A-Hash[%s], B-Hash[%s], %s,%s",
//			to.id(), from.id(), to.tailHeight, to.tailHash, tailBlock.Hash(), from.Details(), to.Details())
//	}
//}
func (self *chainPool) getCurrentBlocks(begin uint64, end uint64) (blocks []commonBlock) {
	for i := begin; i <= end; i++ {
		block := self.tree.Main().GetKnot(i, true)
		if block == nil {
			return
		}
	}
	return blocks
}
