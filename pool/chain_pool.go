package pool

import (
	"fmt"
	"strconv"
	"sync/atomic"

	"sync"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
)

type chainPool struct {
	poolId          string
	log             log15.Logger
	lastestChainIdx int32
	current         *forkedChain
	snippetChains   map[string]*snippetChain // head is fixed
	chains          map[string]*forkedChain
	diskChain       *diskChain

	chainMu sync.Mutex
}

func (self *chainPool) forkChain(forked *forkedChain, snippet *snippetChain) (*forkedChain, error) {
	new := &forkedChain{}

	new.heightBlocks = snippet.heightBlocks
	new.tailHeight = snippet.tailHeight
	new.tailHash = snippet.tailHash
	new.headHeight = snippet.headHeight
	new.headHash = snippet.headHash
	new.referChain = forked

	new.chainId = self.genChainId()

	self.addChain(new)
	return new, nil
}

func (self *chainPool) forkFrom(forked *forkedChain, height uint64, hash types.Hash) (*forkedChain, error) {
	if height == forked.headHeight && hash == forked.headHash {
		return forked, nil
	}
	new := &forkedChain{}

	block := forked.getBlock(height, true)
	if block == nil {
		return nil, errors.New("block is not exist")
	}
	new.init(block)
	new.referChain = forked
	new.chainId = self.genChainId()
	self.addChain(new)
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
	initBlock := self.diskChain.Head()
	self.current.init(initBlock)
	self.current.referChain = self.diskChain
	self.chains = make(map[string]*forkedChain)
	self.snippetChains = make(map[string]*snippetChain)
	self.addChain(self.current)
}

func (self *chainPool) currentModifyToChain(chain *forkedChain) error {
	if chain.id() == self.current.id() {
		return nil
	}
	head := self.diskChain.Head()
	w := chain.getBlock(head.Height(), true)
	if w == nil ||
		w.Hash() != head.Hash() {
		return errors.New("chain can't refer to disk head")
	}
	chain.getBlock(self.current.tailHeight, true)
	if w == nil || w.Hash() != self.current.tailHash {
		return errors.New(fmt.Sprintf("chain tail height error. tailHeight:%d, headHeight:%d", self.current.tailHeight, head.Height()))
	}

	e := self.check()
	if e != nil {
		self.log.Error("---------[1]", "err", e)
	}
	//// todo other chain refer to current ???
	for chain.referChain.id() != self.diskChain.id() {
		fromChain := chain.referChain.(*forkedChain)
		if fromChain.size() == 0 {
			self.log.Error("modify refer[6]", "from", fromChain.id(), "to", chain.id(),
				"fromTailHeight", fromChain.tailHeight, "fromHeadHeight", fromChain.headHeight,
				"toTailHeight", chain.tailHeight, "toHeadHeight", chain.headHeight)
			chain.referChain = fromChain.referChain
			self.modifyChainRefer2(fromChain, chain)
			self.delChain(fromChain.id())
			continue
		}
		e := self.modifyRefer(fromChain, chain)
		if e != nil {
			self.log.Error(e.Error())
			break
		}
		r := clearChainBase(chain)
		if len(r) > 0 {
			self.log.Debug("currentModifyToChain[2]-clearChainBase", "chainId", chain.id(), "start", r[0].Height(), "end", r[len(r)-1].Height())
		}
	}
	self.log.Warn("current modify.", "from", self.current.id(), "to", chain.id(),
		"fromTailHeight", self.current.tailHeight, "fromHeadHeight", self.current.headHeight,
		"toTailHeight", chain.tailHeight, "toHeadHeight", chain.headHeight)
	self.current = chain
	e = self.check()
	if e != nil {
		self.log.Error("---------[2]", "err", e)
	}
	//self.modifyChainRefer()
	return nil
}

func (self *chainPool) modifyRefer(from *forkedChain, to *forkedChain) error {
	r := clearChainBase(to)
	if len(r) > 0 {
		self.log.Debug("modifyRefer-clearChainBase", "chainId", to.id(), "start", r[0].Height(), "end", r[len(r)-1].Height())
	}
	// from.tailHeight <= to.tailHeight  && from.headHeight > to.tail.Height
	toTailHeight := to.tailHeight
	fromTailHeight := from.tailHeight
	fromHeadHeight := from.headHeight
	if fromTailHeight <= toTailHeight && fromHeadHeight > toTailHeight {
		for i := toTailHeight; i > fromTailHeight; i-- {
			w := from.getBlock(i, false)
			if w != nil {
				to.addTail(w)
			}
		}
		for i := fromTailHeight + 1; i <= toTailHeight; i++ {
			w := from.getBlock(i, false)
			if w != nil {
				from.removeTail(w)
			}
		}

		to.referChain = from.referChain
		from.referChain = to

		e := self.modifyChainRefer2(from, to)
		self.log.Info("modify refer", "from", from.id(), "to", to.id(),
			"fromTailHeight", fromTailHeight, "fromHeadHeight", fromHeadHeight,
			"toTailHeight", toTailHeight, "toHeadHeight", to.headHeight, "err", e)
		return nil
	} else {
		return errors.Errorf("err for modifyRefer.from:%s, to:%s, fromTailHeight:%d, fromHeadHeight:%d, toTailHeight:%d, toHeadHeight:%d",
			from.id(), to.id(), fromTailHeight, fromHeadHeight, toTailHeight, to.headHeight)

	}
}
func (self *chainPool) modifyChainRefer2(from *forkedChain, to *forkedChain) error {
	toTailHeight := to.tailHeight
	fromTailHeight := from.tailHeight
	fromHeadHeight := from.headHeight
	for _, v := range self.allChain() {
		if v.id() == from.id() || v.id() == to.id() {
			continue
		}
		if v.referChain.id() == from.id() {
			block, reader := to.getBlockByChain(v.tailHeight)
			if block != nil && block.Hash() == v.tailHash {
				if reader.id() == self.diskChain.id() {
					self.log.Info("modify refer[7]", "from", from.id(), "to", to.id(),
						"fromTailHeight", fromTailHeight, "fromHeadHeight", fromHeadHeight,
						"toTailHeight", toTailHeight, "toHeadHeight", to.headHeight,
						"v", v.id(),
						"vTailHeight", v.tailHeight, "vTailHash", v.tailHash)
					v.referChain = to
				} else {
					self.log.Info("modify refer[8]", "from", from.id(), "to", to.id(),
						"fromTailHeight", fromTailHeight, "fromHeadHeight", fromHeadHeight,
						"toTailHeight", toTailHeight, "toHeadHeight", to.headHeight,
						"v", v.id(),
						"vTailHeight", v.tailHeight, "vTailHash", v.tailHash,
						"readerId", reader.id())
					v.referChain = reader
				}
			} else {
				self.log.Warn("modify refer[4]", "from", from.id(), "to", to.id(),
					"fromTailHeight", fromTailHeight, "fromHeadHeight", fromHeadHeight,
					"toTailHeight", toTailHeight, "toHeadHeight", to.headHeight,
					"v", v.id(),
					"vTailHeight", v.tailHeight, "vTailHash", v.tailHash)
			}
		}
	}
	return nil
}

func (self *chainPool) modifyChainRefer() {
	cs := self.allChain()
	for _, c := range cs {
		if c.id() == self.current.id() {
			continue
		}
		b, reader := c.referChain.getBlockByChain(c.tailHeight)
		if b != nil {
			if reader.id() == self.diskChain.id() {
				if c.referChain.id() == self.current.id() {
					continue
				}
				c.referChain = self.current
				self.log.Warn("[1]modify for refer.", "from", c.id(), "refer", c.referChain.id(), "tailHeight", c.tailHeight)
				continue
			}

			if reader.id() == c.referChain.id() {
				continue
			}

			if c.id() == reader.id() {
				self.log.Error("err for modifyChainRefer refer self.", "from", c.id(), "refer", c.referChain.id(), "tailHeight", c.tailHeight)
				continue
			}
			self.log.Warn("[2]modify for refer.", "from", c.id(), "refer", c.referChain.id(), "tailHeight", c.tailHeight)
			c.referChain = reader
		} else {
			self.log.Error("err for modifyChainRefer.", "from", c.id(), "refer", c.referChain.id(), "tailHeight", c.tailHeight)
			for _, v := range cs {
				b2, r2 := v.getBlockByChain(c.tailHeight)
				if b2 != nil {
					if r2.id() == self.diskChain.id() {
						self.log.Warn("[3]modify for refer.", "from", c.id(), "refer", c.referChain.id(), "tailHeight", c.tailHeight)
						c.referChain = self.current
						break
					}
					if r2.id() == c.id() {
						self.log.Error("err for modifyChainRefer refer self r2.", "from", c.id(), "refer", c.referChain.id(), "tailHeight", c.tailHeight)
						continue
					}
					self.log.Warn("[4]modify for refer.", "from", c.id(), "refer", c.referChain.id(), "tailHeight", c.tailHeight)
					c.referChain = r2
					break
				}
			}
		}
	}
}

func (self *chainPool) currentModify(initBlock commonBlock) {
	new := &forkedChain{}
	new.chainId = self.genChainId()
	new.init(initBlock)
	new.referChain = self.diskChain
	self.addChain(new)
	c := self.current
	c.referChain = new
	self.current = new
	self.modifyChainRefer2(c, new)
}
func (self *chainPool) fork2(snippet *snippetChain, chains []*forkedChain) (bool, bool, *forkedChain) {

	var forky, insertable bool
	var result *forkedChain = nil
	var hr heightChainReader

	trace := ""

LOOP:
	for _, c := range chains {
		tH := snippet.tailHeight
		tHash := snippet.tailHash
		block, reader := c.getBlockByChain(tH)
		if block == nil || block.Hash() != tHash {
			continue
		}

		for i := tH + 1; i <= snippet.headHeight; i++ {
			trace = ""
			b2, r2 := c.getBlockByChain(i)
			sb := snippet.getBlock(i)
			if b2 == nil {
				forky = false
				insertable = true
				hr = reader
				trace += "[1]"
				break LOOP
			}
			if b2.Hash() != sb.Hash() {
				if r2.id() == reader.id() {
					forky = true
					insertable = false
					hr = reader
					trace += "[2]"
					break LOOP
				}

				rhead := reader.Head()
				if rhead.Height() == tH && rhead.Hash() == tHash {
					forky = false
					insertable = true
					hr = reader
					trace += "[3]"
					break LOOP
				}
			} else {
				reader = r2
				block = b2
				tail := snippet.remTail()
				if tail == nil {
					delete(self.snippetChains, snippet.id())
					hr = nil
					trace += "[4]"
					break LOOP
				}
				tH = tail.Height()
				tHash = tail.Hash()
			}
		}
	}

	if hr == nil {
		return false, false, nil
	}
	switch t := hr.(type) {
	case *diskChain:
		result = self.current
		if insertable {
			if self.current.headHeight == snippet.tailHeight && self.current.headHash == snippet.tailHash {
				forky = false
				insertable = true
				result = self.current
				trace += "[5]"
			} else {
				forky = true
				insertable = false
				result = self.current
				trace += "[6]"
			}
		}
	case *forkedChain:
		trace += "[7]"
		result = t
	}
	if insertable {
		err := result.canAddHead(snippet.getBlock(snippet.tailHeight + 1))
		if err != nil {
			self.log.Error("fork2 fail.",
				"sTailHeight", snippet.tailHeight, "sHeadHeight",
				snippet.headHeight, "cTailHeight", result.tailHeight, "cHeadHeight", result.headHeight,
				"trace", trace)
		}
	}

	return forky, insertable, result
}

func (self *chainPool) forky(snippet *snippetChain, chains []*forkedChain) (bool, bool, *forkedChain) {
	for _, c := range chains {
		tailHeight := snippet.tailHeight
		tailHash := snippet.tailHash
		if tailHeight == c.headHeight && tailHash == c.headHash {
			return false, true, c
		}
		//bHeight <= c.tailHeight
		if tailHeight > c.headHeight {
			continue
		}
		// forky
		targetTailBlock := c.getBlock(tailHeight, true)
		if targetTailBlock != nil && targetTailBlock.Hash() == tailHash {
			// same chain
			if sameChain(snippet, c) {
				cutSnippet(snippet, c.headHeight)
				if snippet.headHeight == snippet.tailHeight {
					delete(self.snippetChains, snippet.id())
					return false, false, nil
				} else {
					return false, true, c
				}
			}
			// fork point
			point := findForkPoint(snippet, c, false)
			if point != nil {
				return true, false, c
			}
		}
		if snippet.headHeight == snippet.tailHeight {
			delete(self.snippetChains, snippet.id())
			return false, false, nil
		}
	}

	if snippet.tailHeight <= self.diskChain.Head().Height() {
		point := findForkPoint(snippet, self.current, true)
		if point != nil {
			return true, false, self.current
		}
	}
	// todo duplication code
	if snippet.headHeight == snippet.tailHeight {
		delete(self.snippetChains, snippet.id())
		return false, false, nil
	}
	return false, false, nil
}

func (self *chainPool) insertSnippet(c *forkedChain, snippet *snippetChain) error {
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
func (self *chainPool) insert(c *forkedChain, wrapper commonBlock) error {
	if wrapper.Height() == c.headHeight+1 {
		if c.headHash == wrapper.PrevHash() {
			c.addHead(wrapper)
			return nil
		} else {
			return errors.Errorf("forkedChain fork, fork point height[%d],hash[%s], but next block[%s]'s preHash is [%s]",
				c.headHeight, c.headHash, wrapper.Hash(), wrapper.PrevHash())
		}
	} else {
		return errors.Errorf("forkedChain fork, fork point height[%d],hash[%s], but next block[%s]'s preHash[%s]-[%d]",
			c.headHeight, c.headHash, wrapper.Hash(), wrapper.PrevHash(), wrapper.Height())
	}
}

func (self *chainPool) insertNotify(head commonBlock) {
	if self.current.headHeight == self.current.tailHeight {
		if self.current.tailHash == head.PrevHash() && self.current.headHash == head.PrevHash() {
			self.current.headHash = head.Hash()
			self.current.tailHash = head.Hash()
			self.current.tailHeight = head.Height()
			self.current.headHeight = head.Height()
			return
		}
	}
	self.currentModify(head)
}

func (self *chainPool) writeToChain(chain *forkedChain, block commonBlock) error {
	height := block.Height()
	hash := block.Hash()
	err := self.diskChain.rw.insertBlock(block)
	if err == nil {
		chain.removeTail(block)
		//self.fixReferInsert(chain, self.diskChain, height)
		return nil
	} else {
		self.log.Error(fmt.Sprintf("waiting pool insert forkedChain fail. height:[%d], hash:[%s]", height, hash))
		return err
	}
}

func (self *chainPool) writeBlocksToChain(chain *forkedChain, blocks []commonBlock) error {
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
		chain.removeTail(b)
	}
	return nil
}
func (self *chainPool) check() error {
	diskId := self.diskChain.id()
	currentId := self.current.id()
	for _, c := range self.allChain() {
		// refer to disk
		if c.referChain.id() == diskId {
			if c.id() != currentId {
				self.log.Error(fmt.Sprintf("chain:%s, refer disk.", c.id()))
				return errors.New("refer disk")
			} else {
				err := checkHeadTailLink(c, c.referChain)
				if err != nil {
					self.log.Error(err.Error())
					return err
				}
			}
		} else if c.referChain.id() == currentId {
			// refer to current
			err := checkLink(c, c.referChain, true)
			if err != nil {
				self.log.Error(err.Error())
				return err
			}
		} else {
			err := checkLink(c, c.referChain, false)
			if err != nil {
				self.log.Error(err.Error())
				return err
			}
		}
	}
	return nil
}
func (self *chainPool) addChain(c *forkedChain) {
	self.chainMu.Lock()
	defer self.chainMu.Unlock()
	self.chains[c.id()] = c
}
func (self *chainPool) getChain(id string) *forkedChain {
	self.chainMu.Lock()
	defer self.chainMu.Unlock()
	return self.chains[id]
}
func (self *chainPool) allChain() []*forkedChain {
	self.chainMu.Lock()
	defer self.chainMu.Unlock()
	return copyChains(self.chains)
}
func (self *chainPool) clearUselessChain() []*forkedChain {
	self.chainMu.Lock()
	defer self.chainMu.Unlock()

	var r []*forkedChain
	for id, c := range self.chains {
		if id == self.current.id() {
			continue
		}
		if c.size() == 0 {
			delete(self.chains, id)
			r = append(r, c)
		}
	}
	return r
}
func (self *chainPool) clearRepeatChain() []*forkedChain {
	self.chainMu.Lock()
	defer self.chainMu.Unlock()

	var r []*forkedChain
	for id1, c1 := range self.chains {
		if id1 == self.current.id() {
			continue
		}
		for _, c2 := range self.chains {
			if c1.tailHeight == c2.tailHeight &&
				c1.tailHash == c2.tailHash {
				h := c2.getHeightBlock(c1.headHeight)
				if h != nil && h.Hash() == c1.headHash {
					r = append(r, c1)
				}
			}
		}
	}
	return r
}
func (self *chainPool) delChain(id string) {
	self.chainMu.Lock()
	defer self.chainMu.Unlock()
	delete(self.chains, id)
}
func (self *chainPool) size() int {
	self.chainMu.Lock()
	defer self.chainMu.Unlock()
	return len(self.chains)
}
func (self *chainPool) findOtherChainsByTail(cur *forkedChain, hash types.Hash, height uint64) []*forkedChain {
	self.chainMu.Lock()
	defer self.chainMu.Unlock()
	var result []*forkedChain
	for _, v := range self.chains {
		if v.id() == cur.id() {
			continue
		}
		if v.tailHeight == height {
			if v.tailHash == hash {
				result = append(result, v)
			}
			continue
		}
		b := v.getHeightBlock(height)
		if b != nil && b.Hash() == hash && v.tailHeight < height {
			result = append(result, v)
			continue
		}
	}
	return result
}
