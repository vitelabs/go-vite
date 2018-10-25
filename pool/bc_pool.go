package pool

import (
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
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
	getBlockByChain(height uint64) (commonBlock, heightChainReader)
	Head() commonBlock
	refer() heightChainReader
}

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

	rstat *recoverStat
}

type blockPool struct {
	freeBlocks     map[types.Hash]commonBlock // free state
	compoundBlocks map[types.Hash]commonBlock // compound state
	pendingMu      sync.Mutex
}

type diskChain struct {
	rw      chainRw
	chainId string
	v       *ForkVersion
}

func (self *diskChain) getBlockByChain(height uint64) (commonBlock, heightChainReader) {
	block := self.getBlock(height, false)
	if block != nil {
		return block, self
	} else {
		return nil, nil
	}
}

func (self *diskChain) refer() heightChainReader {
	return nil
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
func (self *diskChain) info() map[string]interface{} {
	result := make(map[string]interface{})
	head := self.rw.head()
	result["Head"] = fmt.Sprintf("%s-%d", head.Hash().String(), head.Height())
	result["Id"] = self.id()
	return result

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
func (self *chain) detail() map[string]interface{} {
	blocks := copyValues(self.heightBlocks)
	sort.Sort(ByHeight(blocks))
	result := make(map[string]interface{})
	var bList []string
	for _, v := range blocks {
		bList = append(bList, fmt.Sprintf("%d-%s-%s", v.Height(), v.Hash(), v.PrevHash()))
	}
	result["Blocks"] = bList
	return result
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

func (self *snippetChain) info() map[string]interface{} {
	result := make(map[string]interface{})
	result["TailHeight"] = self.tailHeight
	result["TailHash"] = self.tailHash
	result["HeadHeight"] = self.headHeight
	result["HeadHash"] = self.headHash
	result["Id"] = self.id()
	return result
}
func (self *snippetChain) getBlock(height uint64) commonBlock {
	block, ok := self.heightBlocks[height]
	if ok {
		return block
	} else {
		return nil
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

func (self *forkedChain) getBlockByChain(height uint64) (commonBlock, heightChainReader) {
	if height > self.headHeight {
		return nil, nil
	}
	block := self.getHeightBlock(height)
	if block != nil {
		return block, self
	}
	refers := make(map[string]heightChainReader)
	refer := self.referChain
	for {
		if refer == nil {
			return nil, nil
		}
		b := refer.getBlock(height, false)
		if b != nil {
			return b, refer
		} else {
			if _, ok := refers[refer.id()]; ok {
				monitor.LogEvent("pool", "getBlockError")
				return nil, nil
			}
			refers[refer.id()] = refer
			refer = refer.refer()
		}
	}
	return nil, nil
}

func (self *forkedChain) refer() heightChainReader {
	return self.referChain
}

func (self *forkedChain) getBlock(height uint64, flag bool) commonBlock {
	block := self.getHeightBlock(height)
	if block != nil {
		return block
	}
	if flag {
		b, _ := self.getBlockByChain(height)
		return b
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

func (self *forkedChain) info() map[string]interface{} {
	result := make(map[string]interface{})
	result["TailHeight"] = self.tailHeight
	result["TailHash"] = self.tailHash
	result["HeadHeight"] = self.headHeight
	result["HeadHash"] = self.headHash
	if self.referChain != nil {
		result["ReferId"] = self.referChain.id()
	}
	result["Id"] = self.id()
	return result
}

func newBlockChainPool(name string) *BCPool {
	return &BCPool{
		Id: name,
	}
}
func (self *BCPool) init(tools *tools) {
	self.tools = tools
	self.compactLock = &common.NonBlockLock{}

	self.LIMIT_HEIGHT = 60 * 60
	self.LIMIT_LONGEST_NUM = 4
	self.rstat = (&recoverStat{}).reset(10)
	self.initPool()
}

func (self *BCPool) initPool() {
	diskChain := &diskChain{chainId: self.Id + "-diskchain", rw: self.tools.rw, v: self.version}
	chainpool := &chainPool{
		poolId:    self.Id,
		diskChain: diskChain,
		log:       self.log,
	}
	chainpool.current = &forkedChain{}
	chainpool.current.chainId = chainpool.genChainId()
	chainpool.init()
	blockpool := &blockPool{
		compoundBlocks: make(map[types.Hash]commonBlock),
		freeBlocks:     make(map[types.Hash]commonBlock),
	}
	self.chainpool = chainpool
	self.blockpool = blockpool
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
	if snippet.headHeight >= head.Height() {
		b := snippet.heightBlocks[head.Height()]
		if b != nil && b.Hash() == head.Hash() {
			return true
		} else {
			return false
		}
	} else {
		b := chain.getBlock(snippet.headHeight, true)
		if b != nil && b.Hash() == snippet.headHash {
			return true
		} else {
			return false
		}
	}
}

// snippet.tailHeight <= chain.headHeight
func findForkPoint(snippet *snippetChain, chain heightChainReader, refer bool) commonBlock {
	tailHeight := snippet.tailHeight
	headHeight := snippet.headHeight

	start := tailHeight + 1
	forkpoint := chain.getBlock(start, refer)
	if forkpoint == nil {
		return nil
	}
	if forkpoint.PrevHash() != snippet.tailHash {
		return nil
	}

	for i := start + 1; i <= headHeight; i++ {
		uncle := chain.getBlock(i, refer)
		if uncle == nil {
			break
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
	return forkpoint
}

func (self *BCPool) rollbackCurrent(blocks []commonBlock) error {
	if len(blocks) <= 0 {
		return nil
	}

	// from small to big
	sort.Sort(ByHeight(blocks))

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
	err := self.checkChain(blocks)
	if err != nil {
		return err
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

func checkHeadTailLink(c1 *forkedChain, c2 heightChainReader) error {
	head := c2.Head()
	if head == nil {
		return errors.New(fmt.Sprintf("checkHeadTailLink fail. c1:%s, c2:%s, head is nil", c1.id(), c2.id()))
	}
	if head.Height() != c1.tailHeight || head.Hash() != c1.tailHash {
		return errors.New(fmt.Sprintf("checkHeadTailLink fail. c1:%s, c2:%s, tailHeight:%d, headHeight:%d, tailHash:%s, headHash:%s", c1.id(), c2.id(), c1.tailHeight, head.Height(), c1.tailHash, head.Hash()))
	}
	return nil
}
func checkLink(c1 *forkedChain, c2 heightChainReader, refer bool) error {
	tailHeight := c1.tailHeight
	block := c2.getBlock(tailHeight, refer)
	if block == nil {
		c2.getBlock(tailHeight, refer)
		return errors.New(fmt.Sprintf("checkLink fail. c1:%s, c2:%s, refer:%t, tailHeight:%d", c1.id(), c2.id(), refer, tailHeight))
	} else if block.Hash() != c1.tailHash {
		c2.getBlock(tailHeight, refer)
		return errors.New(fmt.Sprintf("checkLink fail. c1:%s, c2:%s, refer:%t, tailHeight:%d, tailHash:%s, blockHash:%s", c1.id(), c2.id(), refer, tailHeight, c1.tailHash.String(), block.Hash().String()))
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

func (self *forkedChain) canAddHead(w commonBlock) error {
	if w.Height() == self.headHeight+1 {
		if self.headHash == w.PrevHash() {
			return nil
		}
	}
	return errors.Errorf("can't add head. c.headH:%d-%s, w.prevH:%d-%s", self.headHeight, self.headHash, w.Height()-1, w.PrevHash())
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

func (self *forkedChain) removeHead(w commonBlock) {
	self.headHash = w.PrevHash()
	self.headHeight = w.Height() - 1
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

func (self *blockPool) get(hash types.Hash) commonBlock {
	b1, free := self.freeBlocks[hash]
	if free {
		return b1
	}
	b2, compound := self.compoundBlocks[hash]
	if compound {
		return b2
	}
	return nil
}
func (self *blockPool) containsHash(hash types.Hash) bool {
	_, free := self.freeBlocks[hash]
	_, compound := self.compoundBlocks[hash]
	return free || compound
}

func (self *blockPool) putBlock(hash types.Hash, pool commonBlock) {
	self.freeBlocks[hash] = pool
}
func (self *blockPool) compound(w commonBlock) {
	self.pendingMu.Lock()
	defer self.pendingMu.Unlock()
	self.compoundBlocks[w.Hash()] = w
	delete(self.freeBlocks, w.Hash())
}
func (self *blockPool) afterInsert(w commonBlock) {
	self.pendingMu.Lock()
	defer self.pendingMu.Unlock()
	delete(self.compoundBlocks, w.Hash())
}

func (self *blockPool) delFromCompound(ws map[uint64]commonBlock) {
	self.pendingMu.Lock()
	defer self.pendingMu.Unlock()
	for _, b := range ws {
		delete(self.compoundBlocks, b.Hash())
	}
}

func (self *BCPool) AddBlock(block commonBlock) {
	self.blockpool.pendingMu.Lock()
	defer self.blockpool.pendingMu.Unlock()
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
func (self *BCPool) existInPool(hashes types.Hash) bool {
	self.blockpool.pendingMu.Lock()
	defer self.blockpool.pendingMu.Unlock()
	return self.blockpool.containsHash(hashes)
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
	sortPending := copyValuesFrom(self.blockpool.freeBlocks, &self.blockpool.pendingMu)
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

func (self *BCPool) loopAppendChains() int {
	self.chainpool.check()
	if len(self.chainpool.snippetChains) == 0 {
		return 0
	}
	i := 0
	sortSnippets := copyMap(self.chainpool.snippetChains)
	sort.Sort(ByTailHeight(sortSnippets))

	tmpChains := self.chainpool.allChain()

	for _, w := range sortSnippets {
		forky, insertable, c := self.chainpool.fork2(w, tmpChains)
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
	dels := self.chainpool.clearUselessChain()
	for _, c := range dels {
		self.log.Debug("del useless chain", "info", fmt.Sprintf("%+v", c.id()))
		i++
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

func (self *BCPool) CurrentModifyToChain(target *forkedChain, hashH *ledger.HashHeight) error {
	self.log.Debug("CurrentModifyToChain", "id", target.id(), "TailHeight", target.tailHeight, "HeadHeight", target.headHeight)
	clearChainBase(target)
	return self.chainpool.currentModifyToChain(target)
}
func clearChainBase(target *forkedChain) {
	tailH := target.tailHeight
	base := target.referChain

	for i := tailH + 1; i <= target.headHeight; i++ {
		b := target.getBlock(i, false)
		baseB := base.getBlock(i, true)
		if baseB != nil && baseB.Hash() == b.Hash() {
			target.removeTail(b)
		}
	}
}
func (self *BCPool) CurrentModifyToEmpty() error {
	if self.chainpool.current.size() == 0 {
		return nil
	}
	head := self.chainpool.diskChain.Head()
	self.chainpool.currentModify(head)
	return nil
}

func (self *BCPool) LongestChain() *forkedChain {
	readers := self.chainpool.allChain()
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
func (self *BCPool) CurrentChain() *forkedChain {
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
	for _, c := range self.chainpool.allChain() {
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
	self.chainpool.delChain(c.id())
	self.blockpool.delFromCompound(c.heightBlocks)
}
func (self *BCPool) delSnippet(c *snippetChain) {
	delete(self.chainpool.snippetChains, c.id())
	self.blockpool.delFromCompound(c.heightBlocks)
}
func (self *BCPool) info() map[string]interface{} {
	result := make(map[string]interface{})
	bp := self.blockpool
	cp := self.chainpool

	result["FreeSize"] = len(bp.freeBlocks)
	result["CompoundSize"] = len(bp.compoundBlocks)
	result["SnippetSize"] = len(cp.snippetChains)
	result["ChainSize"] = len(cp.chains)
	result["CurrentLen"] = cp.current.size()
	var snippetIds []interface{}
	for _, v := range cp.snippetChains {
		snippetIds = append(snippetIds, v.info())
	}
	result["Snippets"] = snippetIds
	var chainIds []interface{}
	for _, v := range cp.allChain() {
		chainIds = append(chainIds, v.info())
	}
	result["Chains"] = chainIds
	result["Current"] = cp.current.info()
	result["Disk"] = cp.diskChain.info()

	return result
}

func (self *BCPool) detailChain(id string) map[string]interface{} {
	c := self.chainpool.getChain(id)
	if c != nil {
		return c.detail()
	}
	s := self.chainpool.snippetChains[id]
	if s != nil {
		return s.detail()
	}
	return nil
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
func copyValuesFrom(m map[types.Hash]commonBlock, pendingMu *sync.Mutex) []commonBlock {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	var r []commonBlock

	for _, v := range m {
		r = append(r, v)
	}
	return r
}
func copyValues(m map[uint64]commonBlock) []commonBlock {
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
