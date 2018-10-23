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
	if chain.tailHeight < head.Height() {
		return errors.New(fmt.Sprintf("chain tail height error. tailHeight:%d, headHeight:%d", chain.tailHeight, head.Height()))
	}

	//// todo other chain refer to current ???
	for chain.referChain.id() != self.diskChain.id() {
		fromChain := chain.referChain.(*forkedChain)
		e := self.modifyRefer(fromChain, chain)
		if e != nil {
			self.log.Error(e.Error())
			break
		}
	}
	self.log.Warn("current modify.", "from", self.current.id(), "to", chain.id(),
		"fromTailHeight", self.current.tailHeight, "fromHeadHeight", self.current.headHeight,
		"toTailHeight", chain.tailHeight, "toHeadHeight", chain.headHeight)
	self.current = chain
	self.modifyChainRefer()
	return nil
}

func (self *chainPool) modifyRefer(from *forkedChain, to *forkedChain) error {
	// from.tailHeight <= to.tailHeight  && from.headHeight > to.tail.Height

	if from.tailHeight <= to.tailHeight && from.headHeight > to.tailHeight {
		for i := to.tailHeight; i > from.tailHeight; i-- {
			w := from.getBlock(i, false)
			if w != nil {
				to.addTail(w)
			}
		}
		for i := from.tailHeight + 1; i <= to.tailHeight; i++ {
			w := from.getBlock(i, false)
			if w != nil {
				from.removeTail(w)
			}
		}
		to.referChain = from.referChain
		from.referChain = to

		for _, c := range self.allChain() {
			if c.refer() != nil && c.refer().id() == from.id() {
				if !from.contains(c.tailHeight) {
					block, reader := to.referChain.getBlockByChain(c.tailHeight)
					if block != nil {
						if reader.id() == self.diskChain.id() {
							c.referChain = to
						} else {
							c.referChain = reader
						}
					}
				}
			}
		}
		return nil
	} else {
		return errors.Errorf("err for modifyRefer.", "from", from.id(), "to", to.id(),
			"fromTailHeight", from.tailHeight, "fromHeadHeight", from.headHeight,
			"toTailHeight", to.tailHeight, "toHeadHeight", to.headHeight)

	}
}

func (self *chainPool) modifyChainRefer() {
	for _, c := range self.allChain() {
		if c.id() == self.current.id() {
			continue
		}
		b, reader := c.referChain.getBlockByChain(c.tailHeight)
		if b != nil {
			if reader.id() == c.referChain.id() {
				continue
			}
			if reader.id() == self.diskChain.id() {
				c.referChain = self.current
			} else {
				c.referChain = reader
			}
		} else {
			self.log.Error("err for modifyChainRefer.", "from", c.id(), "refer", c.referChain.id(), "tailHeight", c.tailHeight)
		}
	}
}

func (self *chainPool) currentModify(initBlock commonBlock) {
	new := &forkedChain{}
	new.chainId = self.genChainId()
	new.init(initBlock)
	new.referChain = self.diskChain
	self.current = new
	self.addChain(new)
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
			self.log.Warn(fmt.Sprintf("account forkedChain fork, fork point height[%d],hash[%s], but next block[%s]'s preHash is [%s]",
				c.headHeight, c.headHash, wrapper.Hash(), wrapper.PrevHash()))
			return &ForkChainError{What: "fork chain."}
		}
	} else {
		self.log.Warn(fmt.Sprintf("account forkedChain fork, fork point height[%d],hash[%s], but next block[%s]'s preHash is [%s]",
			c.headHeight, c.headHash, wrapper.Hash(), wrapper.PrevHash()))
		return &ForkChainError{What: "fork chain."}
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
func (self *chainPool) check() {
	diskId := self.diskChain.id()
	currentId := self.current.id()
	for _, c := range self.allChain() {
		// refer to disk
		if c.referChain.id() == diskId {
			if c.id() != currentId {
				self.log.Error(fmt.Sprintf("chain:%s, refer disk.", c.id()))
			} else {
				err := checkHeadTailLink(c, c.referChain)
				if err != nil {
					self.log.Error(err.Error())
				}
			}
		} else if c.referChain.id() == currentId {
			// refer to current
			err := checkLink(c, c.referChain, true)
			if err != nil {
				self.log.Error(err.Error())
			}
		} else {
			err := checkLink(c, c.referChain, false)
			if err != nil {
				self.log.Error(err.Error())
			}
		}
	}
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
