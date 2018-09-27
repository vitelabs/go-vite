package pool

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/verifier"
)

type Chain interface {
	HeadHeight() uint64
	ChainId() string
	Head() commonBlock
	GetBlock(height uint64) commonBlock
}

type ChainReader interface {
	Head() commonBlock
	GetBlock(height uint64) commonBlock
}

type heightChainReader interface {
	id() string
	getBlock(height uint64, refer bool) commonBlock
	contains(height uint64) bool
	Head() commonBlock
}

var pendingMu sync.Mutex

type BCPool struct {
	Id  string
	log log15.Logger

	blockpool *blockPool
	chainpool *chainPool
	tools     *tools

	version *ForkVersion
	rMu     sync.Mutex // direct add and loop insert

	compactLock       *common.NonBlockLock // snippet,chain
	LIMIT_HEIGHT      uint64
	LIMIT_LONGEST_NUM uint64
}

type blockPool struct {
	freeBlocks     map[types.Hash]commonBlock // free state
	compoundBlocks map[types.Hash]commonBlock // compound state
}
type chainPool struct {
	poolId          string
	log             log15.Logger
	lastestChainIdx int32
	current         *forkedChain
	snippetChains   map[string]*snippetChain // head is fixed
	chains          map[string]*forkedChain
	diskChain       *diskChain
	//rw          Chain
}

func (self *chainPool) forkChain(forked *forkedChain, snippet *snippetChain) (*forkedChain, error) {
	new := &forkedChain{}

	new.heightBlocks = snippet.heightBlocks
	new.tailHeight = snippet.tailHeight
	new.headHeight = snippet.headHeight
	new.headHash = snippet.headHash
	new.referChain = forked

	new.chainId = self.genChainId()
	self.chains[new.chainId] = new
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
	self.chains[new.chainId] = new
	return new, nil
}

type diskChain struct {
	rw      chainRw
	chainId string
	v       *ForkVersion
}

func (self *diskChain) getBlock(height uint64, refer bool) commonBlock {
	if height < 0 {
		return nil
	}
	block := self.rw.getBlock(height)
	if block == nil {
		return nil
	} else {
		return block
	}
}
func (self *diskChain) getBlockBetween(tail int, head int, refer bool) commonBlock {
	//forkVersion := version.ForkVersion()
	//block := self.rw.GetBlock(height)
	//if block == nil {
	//	return nil
	//} else {
	//	return &PoolBlock{block: block, forkVersion: forkVersion, verifyStat: &BlockVerifySuccessStat{}}
	//}
	return nil
}

func (self *diskChain) contains(height uint64) bool {
	return self.rw.head().Height() >= height
}

func (self *diskChain) id() string {
	return self.chainId
}

func (self *diskChain) Head() commonBlock {
	head := self.rw.head()
	if head == nil {
		return self.rw.getBlock(types.EmptyHeight) // hack implement
	}

	return head
}

type chain struct {
	heightBlocks map[uint64]commonBlock
	headHeight   uint64 //  forkedChain size is zero when headHeight==tailHeight
	tailHeight   uint64
	chainId      string
}

func (self *chain) size() uint64 {
	return self.headHeight - self.tailHeight
}
func (self *chain) HeadHeight() uint64 {
	return self.headHeight
}

func (self *chain) ChainId() string {
	return self.chainId
}

func (self *chain) id() string {
	return self.chainId
}

type snippetChain struct {
	chain
	tailHash types.Hash
	headHash types.Hash
}

func (self *snippetChain) init(w commonBlock) {
	self.heightBlocks = make(map[uint64]commonBlock)
	self.headHeight = w.Height()
	self.headHash = w.Hash()
	self.tailHash = w.PrevHash()
	self.tailHeight = w.Height() - 1
	self.heightBlocks[w.Height()] = w
}

func (self *snippetChain) addTail(w commonBlock) {
	self.tailHash = w.PrevHash()
	self.tailHeight = w.Height() - 1
	self.heightBlocks[w.Height()] = w
}
func (self *snippetChain) deleteTail(newtail commonBlock) {
	self.tailHash = newtail.Hash()
	self.tailHeight = newtail.Height()
	delete(self.heightBlocks, newtail.Height())
}
func (self *snippetChain) remTail() commonBlock {
	newTail := self.heightBlocks[self.tailHeight+1]
	if newTail == nil {
		return nil
	}
	self.deleteTail(newTail)
	return newTail
}

func (self *snippetChain) merge(snippet *snippetChain) {
	self.tailHeight = snippet.tailHeight
	self.tailHash = snippet.tailHash
	for k, v := range snippet.heightBlocks {
		self.heightBlocks[k] = v
	}
}

type forkedChain struct {
	chain
	// key: height
	headHash   types.Hash
	tailHash   types.Hash
	referChain heightChainReader
	heightMu   sync.RWMutex
}

func (self *forkedChain) getBlock(height uint64, refer bool) commonBlock {
	block := self.getHeightBlock(height)
	if block != nil {
		return block
	}
	if refer {
		return self.referChain.getBlock(height, refer)
	}
	return nil
}
func (self *forkedChain) getHeightBlock(height uint64) commonBlock {
	self.heightMu.RLock()
	defer self.heightMu.RUnlock()
	block, ok := self.heightBlocks[height]
	if ok {
		return block
	} else {
		return nil
	}
}
func (self *forkedChain) setHeightBlock(height uint64, b commonBlock) {
	self.heightMu.Lock()
	defer self.heightMu.Unlock()
	if b != nil {
		self.heightBlocks[height] = b
	} else {
		// nil means delete
		delete(self.heightBlocks, height)
	}
}

func (self *forkedChain) Head() commonBlock {
	return self.GetBlock(self.headHeight)
}

func (self *forkedChain) GetBlock(height uint64) commonBlock {
	w := self.getBlock(height, true)
	if w == nil {
		return nil
	}
	return w
}

func (self *forkedChain) contains(height uint64) bool {
	return height > self.tailHeight && self.headHeight <= height
}

func newBlockChainPool(name string) *BCPool {
	return &BCPool{
		Id: name,
	}
}
func (self *BCPool) init(rw chainRw,
	tools *tools) {

	diskChain := &diskChain{chainId: self.Id + "-diskchain", rw: rw, v: self.version}
	chainpool := &chainPool{
		poolId:    self.Id,
		diskChain: diskChain,
		log:       self.log,
	}
	chainpool.current = &forkedChain{}

	chainpool.current.chainId = chainpool.genChainId()
	blockpool := &blockPool{
		compoundBlocks: make(map[types.Hash]commonBlock),
		freeBlocks:     make(map[types.Hash]commonBlock),
	}
	self.chainpool = chainpool
	self.blockpool = blockpool
	self.tools = tools
	self.compactLock = &common.NonBlockLock{}

	self.chainpool.init()

	self.LIMIT_HEIGHT = 60 * 60
	self.LIMIT_LONGEST_NUM = 4
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
	self.chains[self.current.chainId] = self.current
}

func (self *chainPool) currentModifyToChain(chain *forkedChain) error {
	if chain.id() == self.current.id() {
		return nil
	}
	head := self.diskChain.Head()
	w := chain.getBlock(head.Height(), true)
	if w == nil ||
		w.Hash() != head.Hash() {
		return errors.New("error")
	}

	// todo other chain refer to current ???
	for chain.referChain.id() != self.diskChain.id() {
		fromChain := chain.referChain.(*forkedChain)
		self.modifyRefer(fromChain, chain)
	}
	self.log.Warn("current modify.", "from", self.current.id(), "to", chain.id())
	self.current = chain
	return nil
}

func (self *chainPool) modifyRefer(from *forkedChain, to *forkedChain) {
	for i := to.tailHeight; i > from.tailHeight; i-- {
		w := from.getBlock(i, false)
		from.removeTail(w)
		to.addTail(w)
	}
	to.referChain = from.referChain
	from.referChain = to
}

func (self *chainPool) currentModify(initBlock commonBlock) {
	new := &forkedChain{}
	new.chainId = self.genChainId()
	new.init(initBlock)
	new.referChain = self.diskChain
	self.current = new
	self.chains[new.chainId] = new
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
		if snippet.headHeight <= c.headHeight {
			continue
		}
		if sameChain(snippet, c) {
			cutSnippet(snippet, c.headHeight)
			if snippet.headHeight == snippet.tailHeight {
				delete(self.snippetChains, snippet.id())
				return false, false, nil
			} else {
				return false, true, c
			}
		}
		point := findForkPoint(snippet, c, false)
		if point != nil {
			return true, false, c
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
func cutSnippet(snippet *snippetChain, height uint64) {
	for {
		tail := snippet.remTail()
		if tail == nil {
			return
		}
		if tail.Height() >= height {
			return
		}
	}
}

// snippet.headHeight >=chain.headHeight
func sameChain(snippet *snippetChain, chain heightChainReader) bool {
	head := chain.Head()
	b := snippet.heightBlocks[head.Height()]
	if b != nil && b.Hash() == head.Hash() {
		return true
	} else {
		return false
	}
}

// snippet.tailHeight <= chain.headHeight
func findForkPoint(snippet *snippetChain, chain heightChainReader, refer bool) commonBlock {
	tailHeight := snippet.tailHeight
	headHeight := snippet.headHeight

	forkpoint := chain.getBlock(tailHeight, refer)
	if forkpoint == nil {
		return nil
	}
	if forkpoint.Hash() != snippet.tailHash {
		return nil
	}

	for i := tailHeight + 1; i <= headHeight; i++ {
		uncle := chain.getBlock(i, refer)
		if uncle == nil {
			log15.Error(fmt.Sprintf("chain error. chain:%s", chain))
			return nil
		}
		point := snippet.heightBlocks[i]
		if point.Hash() != uncle.Hash() {
			return forkpoint
		} else {
			snippet.deleteTail(point)
			forkpoint = point
			continue
		}
	}
	return nil
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

func (self *BCPool) rollbackCurrent(blocks []commonBlock) error {
	// from small to big
	sort.Sort(ByHeight(blocks))
	err := self.checkChain(blocks)
	if err != nil {
		return err
	}

	head := self.chainpool.diskChain.Head()
	h := len(blocks) - 1
	smallest := blocks[0]
	longest := blocks[h]
	if head.Height()+1 != smallest.Height() || head.Hash() != smallest.PrevHash() {
		return errors.New(self.Id + " disk chain height hash check fail")
	}

	if self.chainpool.current.tailHeight != longest.Height() || self.chainpool.current.tailHash != longest.Hash() {
		return errors.New(self.Id + " current chain height hash check fail")
	}
	for i := h; i >= 0; i-- {
		self.chainpool.current.addTail(blocks[i])
	}
	return nil
}

// check blocks is a chain
func (self *BCPool) checkChain(blocks []commonBlock) error {
	smalletHeight := blocks[0].Height()

	for i, b := range blocks {
		if b.Height() != smalletHeight+uint64(i) {
			return errors.New("not a chain")
		}
	}
	return nil
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

//func (self *chainPool) fixReferInsert(origin heightChainReader, target heightChainReader, fixHeight int) {
//	originId := origin.id()
//	for id, chain := range self.chains {
//		if chain.referChain.id() == originId && chain.tailHeight <= fixHeight {
//			chain.referChain = target
//			log.Info("forkedChain[%s] reset refer from %s because of %d, refer to disk.", id, originId, fixHeight)
//		}
//	}
//}
//
//func (self *chainPool) fixReferRollback(origin heightChainReader, target heightChainReader, fixHeight int) {
//	originId := origin.id()
//	for id, chain := range self.chains {
//		if chain.referChain.id() == originId && chain.tailHeight> fixHeight {
//			chain.referChain = self.diskChain
//			log.Info("forkedChain[%s] reset refer from %s because of %d, refer to disk.", id, targetId, fixHeight)
//		}
//	}
//}

func (self *forkedChain) init(initBlock commonBlock) {
	self.heightBlocks = make(map[uint64]commonBlock)
	self.tailHeight = initBlock.Height()
	self.tailHash = initBlock.Hash()
	self.headHeight = initBlock.Height()
	self.headHash = initBlock.Hash()
}

func (self *forkedChain) addHead(w commonBlock) {
	self.headHash = w.Hash()
	self.headHeight = w.Height()
	self.setHeightBlock(w.Height(), w)
}

func (self *forkedChain) removeTail(w commonBlock) {
	self.tailHash = w.Hash()
	self.tailHeight = w.Height()
	self.setHeightBlock(w.Height(), nil)
}

func (self *forkedChain) addTail(w commonBlock) {
	self.tailHash = w.PrevHash()
	self.tailHeight = w.Height() - 1
	self.setHeightBlock(w.Height(), w)
}

func (self *forkedChain) String() string {
	return "chainId:\t" + self.chainId + "\n" +
		"headHeight:\t" + strconv.FormatUint(self.headHeight, 10) + "\n" +
		"headHash:\t" + "[" + self.headHash.String() + "]\t" + "\n" +
		"tailHeight:\t" + strconv.FormatUint(self.tailHeight, 10)
}

func (self *blockPool) contains(hash types.Hash, height uint64) bool {
	_, free := self.freeBlocks[hash]
	_, compound := self.compoundBlocks[hash]
	return free || compound
}
func (self *blockPool) putBlock(hash types.Hash, pool commonBlock) {
	self.freeBlocks[hash] = pool
}
func (self *blockPool) compound(w commonBlock) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	self.compoundBlocks[w.Hash()] = w
	delete(self.freeBlocks, w.Hash())
}
func (self *blockPool) afterInsert(w commonBlock) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	delete(self.compoundBlocks, w.Hash())
}

func (self *blockPool) delFromCompound(ws map[uint64]commonBlock) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	for _, b := range ws {
		delete(self.compoundBlocks, b.Hash())
	}
}

func (self *BCPool) AddBlock(block commonBlock) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	hash := block.Hash()
	height := block.Height()
	if !self.blockpool.contains(hash, height) {
		self.blockpool.putBlock(hash, block)
	} else {
		monitor.LogEvent("pool", "addDuplication")
		// todo online del
		self.log.Warn(fmt.Sprintf("block exists in BCPool. hash:[%s], height:[%d].", hash, height))
	}
}

type ByHeight []commonBlock

func (a ByHeight) Len() int           { return len(a) }
func (a ByHeight) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByHeight) Less(i, j int) bool { return a[i].Height() < a[j].Height() }

type ByTailHeight []*snippetChain

func (a ByTailHeight) Len() int           { return len(a) }
func (a ByTailHeight) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTailHeight) Less(i, j int) bool { return a[i].tailHeight < a[j].tailHeight }

func (self *BCPool) loopGenSnippetChains() int {
	if len(self.blockpool.freeBlocks) == 0 {
		return 0
	}

	i := 0
	//  self.chainpool.snippetChains
	// todo why copy ?
	sortPending := copyValuesFrom(self.blockpool.freeBlocks)
	sort.Sort(sort.Reverse(ByHeight(sortPending)))

	// todo why copy ?
	chains := copyMap(self.chainpool.snippetChains)

	for _, v := range sortPending {
		if !tryInsert(chains, v) {
			snippet := &snippetChain{}
			snippet.chainId = self.chainpool.genChainId()
			snippet.init(v)
			chains = append(chains, snippet)
		}
		i++
		self.blockpool.compound(v)
	}

	headMap := splitToMap(chains)
	for _, chain := range chains {
		for {
			tail := chain.tailHash
			oc, ok := headMap[tail]
			if ok && chain.id() != oc.id() {
				delete(headMap, tail)
				chain.merge(oc)
			} else {
				break
			}
		}
	}
	final := make(map[string]*snippetChain)
	for _, v := range headMap {
		final[v.id()] = v
	}
	self.chainpool.snippetChains = final
	return i
}
func (self *BCPool) AddDirectBlock(block commonBlock) error {
	self.rMu.Lock()
	defer self.rMu.Unlock()

	stat := self.tools.verifier.verify(block)
	result := stat.verifyResult()
	switch result {
	case verifier.PENDING:
		return errors.New("pending for something")
	case verifier.FAIL:
		return errors.New(stat.errMsg())
	case verifier.SUCCESS:
		err := self.chainpool.diskChain.rw.insertBlock(block)
		if err != nil {
			return err
		}
		head := self.chainpool.diskChain.Head()
		self.chainpool.insertNotify(head)
		return nil
	default:
		self.log.Crit("verify unexpected.")
		return errors.New("verify unexpected")
	}
}
func (self *BCPool) loopAppendChains() int {
	if len(self.chainpool.snippetChains) == 0 {
		return 0
	}
	i := 0
	sortSnippets := copyMap(self.chainpool.snippetChains)
	sort.Sort(ByTailHeight(sortSnippets))

	tmpChains := copyChains(self.chainpool.chains)

	for _, w := range sortSnippets {
		forky, insertable, c := self.chainpool.forky(w, tmpChains)
		if forky {
			i++
			newChain, err := self.chainpool.forkChain(c, w)
			if err == nil {
				tmpChains = append(tmpChains, newChain)
				delete(self.chainpool.snippetChains, w.id())
			}
			continue
		}
		if insertable {
			i++
			err := self.chainpool.insertSnippet(c, w)
			if err != nil {
				self.log.Error("insert fail.", "err", err)
			}
			continue
		}
	}
	return i
}
func (self *BCPool) loopFetchForSnippets() int {
	if len(self.chainpool.snippetChains) == 0 {
		return 0
	}
	sortSnippets := copyMap(self.chainpool.snippetChains)
	sort.Sort(ByTailHeight(sortSnippets))

	head := new(big.Int).SetUint64(self.chainpool.current.headHeight)
	i := 0
	zero := big.NewInt(0)
	prev := zero

	for _, w := range sortSnippets {
		diff := big.NewInt(0)
		tailHeight := new(big.Int).SetUint64(w.tailHeight)
		// prev > 0
		if prev.Sign() > 0 {
			diff.Sub(tailHeight, prev)
		} else {
			// first snippet
			diff.Sub(tailHeight, head)
		}

		// prev and this chains have fork
		if diff.Sign() <= 0 {
			diff.Sub(tailHeight, head)
		}

		// lower than the current chain
		if diff.Sign() <= 0 {
			diff.SetUint64(20)
		}

		i++
		hash := ledger.HashHeight{Hash: w.tailHash, Height: w.tailHeight}
		self.tools.fetcher.fetch(hash, diff.Uint64())

		prev.SetUint64(w.headHeight)
	}
	return i
}

func (self *BCPool) CurrentModifyToChain(target Chain) error {
	chain := target.(*forkedChain)
	return self.chainpool.currentModifyToChain(chain)
}
func (self *BCPool) CurrentModifyToEmpty() error {
	if self.chainpool.current.size() == 0 {
		return nil
	}
	head := self.chainpool.diskChain.Head()
	self.chainpool.currentModify(head)
	return nil
}

func (self *BCPool) LongestChain() Chain {
	readers := self.chainpool.chains
	current := self.chainpool.current
	longest := current
	for _, reader := range readers {
		height := reader.headHeight
		if height > longest.headHeight {
			longest = reader
		}
	}
	if longest.headHeight-self.LIMIT_LONGEST_NUM > current.headHeight {
		return longest
	} else {
		return current
	}
}
func (self *BCPool) CurrentChain() Chain {
	return self.chainpool.current
}

// keyPoint, forkPoint, err
func (self *BCPool) getForkPointByChains(chain1 Chain, chain2 Chain) (commonBlock, commonBlock, error) {
	if chain1.Head().Height() > chain2.Head().Height() {
		return self.getForkPoint(chain1, chain2)
	} else {
		return self.getForkPoint(chain2, chain1)
	}
}

// keyPoint, forkPoint, err
func (self *BCPool) getForkPoint(longest Chain, current Chain) (commonBlock, commonBlock, error) {
	curHeadHeight := current.HeadHeight()

	i := curHeadHeight
	var forkedBlock commonBlock

	for {
		block := longest.GetBlock(i)
		curBlock := current.GetBlock(i)
		if block == nil {
			self.log.Error("longest chain is not longest.", "chainId", longest.ChainId(), "height", i)
			return nil, nil, errors.New("longest chain error.")
		}

		if curBlock == nil {
			self.log.Error("current chain is wrong.", "chainId", current.ChainId(), "height", i)
			return nil, nil, errors.New("current chain error.")
		}

		if block.Hash() == curBlock.Hash() {
			forkedBlock = block
			keyPoint := longest.GetBlock(i + 1)
			return keyPoint, forkedBlock, nil
		}
		i = i - 1
	}
	return nil, nil, errors.New("can't find fork point")
}

func (self *BCPool) loop() {
	for {
		self.loopGenSnippetChains()
		self.loopAppendChains()
		self.loopFetchForSnippets()
		//self.CheckCurrentInsert()
		time.Sleep(time.Second)
	}
}

func (self *BCPool) loopDelUselessChain() {
	// if an insert operation is in progress, do nothing.
	// todo add tryLockWait method
	if !self.compactLock.TryLock() {
		return
	} else {
		defer self.compactLock.UnLock()
	}

	height := self.chainpool.current.tailHeight
	for _, c := range self.chainpool.chains {
		if c.headHeight+self.LIMIT_HEIGHT < height {
			self.delChain(c)
		}
	}
	for _, c := range self.chainpool.snippetChains {
		if c.headHeight+self.LIMIT_HEIGHT < height {
			self.delSnippet(c)
		}
	}
}
func (self *BCPool) delChain(c *forkedChain) {
	delete(self.chainpool.chains, c.id())
	self.blockpool.delFromCompound(c.heightBlocks)
}
func (self *BCPool) delSnippet(c *snippetChain) {
	delete(self.chainpool.snippetChains, c.id())
	self.blockpool.delFromCompound(c.heightBlocks)
}

func splitToMap(chains []*snippetChain) map[types.Hash]*snippetChain {
	headMap := make(map[types.Hash]*snippetChain)
	//tailMap := make(map[string]*snippetChain)
	for _, chain := range chains {
		head := chain.headHash
		//tail := chain.tailHash
		headMap[head] = chain
		//tailMap[tail] = chain
	}
	return headMap
}

func tryInsert(chains []*snippetChain, pool commonBlock) bool {
	for _, c := range chains {
		if c.tailHash == pool.Hash() {
			c.addTail(pool)
			return true
		}
		height := pool.Height()
		if c.headHeight > height && height > c.tailHeight {
			// when block is in snippet
			if c.heightBlocks[pool.Height()].Hash() == pool.Hash() {
				return true
			}
			continue
		}
	}
	return false
}

func copyMap(m map[string]*snippetChain) []*snippetChain {
	var s []*snippetChain
	for _, v := range m {
		s = append(s, v)
	}
	return s
}
func copyValuesFrom(m map[types.Hash]commonBlock) []commonBlock {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	var r []commonBlock

	for _, v := range m {
		r = append(r, v)
	}
	return r
}

func copyChains(m map[string]*forkedChain) []*forkedChain {
	var r []*forkedChain

	for _, v := range m {
		r = append(r, v)
	}
	return r
}
