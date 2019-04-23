package pool

import (
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/common"

	"github.com/vitelabs/go-vite/pool/tree"

	"github.com/pkg/errors"
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

type BCPool struct {
	Id  string
	log log15.Logger

	blockpool *blockPool
	chainpool *chainPool
	tools     *tools

	version *ForkVersion

	// 1. protecting the tail(hash && height) of current chain.
	// 2. protecting the modification for the current chain (which is the current chain?).
	chainTailMu sync.Mutex
	chainHeadMu sync.Mutex

	LIMIT_HEIGHT      uint64
	LIMIT_LONGEST_NUM uint64

	rstat *recoverStat
}

type blockPool struct {
	freeBlocks     map[types.Hash]commonBlock // free state_bak
	compoundBlocks sync.Map                   // compound state_bak
	pendingMu      sync.Mutex
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

//func (self *chain) detail() map[string]interface{} {
//	blocks := copyValues(self.heightBlocks)
//	sort.Sort(ByHeight(blocks))
//	result := make(map[string]interface{})
//	var bList []string
//	for _, v := range blocks {
//		bList = append(bList, fmt.Sprintf("%d-%s-%s", v.Height(), v.Hash(), v.PrevHash()))
//	}
//	result["Blocks"] = bList
//	return result
//}

type snippetChain struct {
	chain
	tailHash types.Hash
	headHash types.Hash
}

func newSnippetChain(w commonBlock, id string) *snippetChain {
	self := &snippetChain{}
	self.chainId = id
	self.heightBlocks = make(map[uint64]commonBlock)
	self.headHeight = w.Height()
	self.headHash = w.Hash()
	self.tailHash = w.PrevHash()
	self.tailHeight = w.Height() - 1
	self.heightBlocks[w.Height()] = w
	return self
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

func (self *BCPool) init(tools *tools) {
	self.tools = tools

	self.LIMIT_HEIGHT = 75 * 2
	self.LIMIT_LONGEST_NUM = 3
	self.rstat = (&recoverStat{}).init(10, 10*time.Second)
	self.initPool()
}

func (self *BCPool) initPool() {
	diskChain := &branchChain{chainId: self.Id + "-diskchain", rw: self.tools.rw, v: self.version}

	t := tree.NewTree()
	chainpool := &chainPool{
		poolId:    self.Id,
		diskChain: diskChain,
		tree:      t,
		log:       self.log,
	}
	chainpool.init()
	blockpool := &blockPool{
		freeBlocks: make(map[types.Hash]commonBlock),
	}
	self.chainpool = chainpool
	self.blockpool = blockpool
}

func (self *BCPool) rollbackCurrent(blocks []commonBlock) error {
	if len(blocks) <= 0 {
		return nil
	}
	cur := self.CurrentChain()
	curTailHeight, curTailHash := cur.TailHH()
	// from small to big
	sort.Sort(ByHeight(blocks))

	self.log.Info("rollbackCurrent", "start", blocks[0].Height(), "end", blocks[len(blocks)-1].Height(), "size", len(blocks),
		"currentId", cur.Id())
	disk := self.chainpool.diskChain
	headHeight, headHash := disk.HeadHH()

	err := self.checkChain(blocks)
	if err != nil {
		self.log.Info("check chain fail." + self.printf(blocks))
		panic(err)
		return err
	}

	if cur.Linked(disk) {
		self.log.Info("poolChain and db is connected.", "headHeight", headHeight, "headHash", headHash)
		return nil
	}

	h := len(blocks) - 1
	smallest := blocks[0]
	longest := blocks[h]
	if headHeight+1 != smallest.Height() || headHash != smallest.PrevHash() {
		for _, v := range blocks {
			self.log.Info("block delete", "height", v.Height(), "hash", v.Hash(), "prevHash", v.PrevHash())
		}
		self.log.Crit("error for db fail.", "headHeight", headHeight, "headHash", headHash, "smallestHeight", smallest.Height(), "err", errors.New(self.Id+" disk chain height hash check fail"))
		return errors.New(self.Id + " disk chain height hash check fail")
	}

	if curTailHeight != longest.Height() || curTailHash != longest.Hash() {
		for _, v := range blocks {
			self.log.Info("block delete", "height", v.Height(), "hash", v.Hash(), "prevHash", v.PrevHash())
		}
		self.log.Crit("error for db fail.", "tailHeight", curTailHeight, "tailHash", curTailHash, "longestHeight", longest.Height(), "err", errors.New(self.Id+" current chain height hash check fail"))
		return errors.New(self.Id + " current chain height hash check fail")
	}
	main := self.chainpool.tree.Main()
	for i := h; i >= 0; i-- {
		err := self.chainpool.tree.AddTail(main, blocks[i])
		if err != nil {
			panic(err)
		}
	}
	err = self.chainpool.check()
	if err != nil {
		self.log.Error("rollbackCurrent check", "err", err)
	}
	return nil
}

// check blocks is a chain
func (self *BCPool) checkChain(blocks []commonBlock) error {
	var prev commonBlock
	for _, b := range blocks {
		if prev == nil {
			prev = b
			continue
		}
		if b.PrevHash() != prev.Hash() {
			return errors.New("not a chain.")
		}
		if b.Height()-1 != prev.Height() {
			return errors.New("not a chain.")
		}
		prev = b
	}
	return nil
}

// check blocks is a chain
func (self *BCPool) printf(blocks []commonBlock) string {
	result := ""
	for _, v := range blocks {
		result += fmt.Sprintf("[%d-%s-%s]", v.Height(), v.Hash(), v.PrevHash())
	}
	return result
}

func checkHeadTailLink(c1 tree.Branch, c2 tree.Branch) error {
	if c1.Linked(c2) {
		return nil
	}
	return errors.New(fmt.Sprintf("checkHeadTailLink fail. c1:%s, c2:%s, c1Tail:%s, c1Head:%s, c2Tail:%s, c2Head:%s",
		c1.Id(), c2.Id(), c1.SprintTail(), c1.SprintHead(), c2.SprintTail(), c2.SprintHead()))
}
func checkLink(c1 tree.Branch, c2 tree.Branch, refer bool) error {
	tailHeight, tailHash := c1.TailHH()
	block := c2.GetKnot(tailHeight, refer)
	if block == nil {
		return errors.New(fmt.Sprintf("checkLink fail. c1:%s, c2:%s, refer:%t, c1Tail:%s, c1Head:%s, c2Tail:%s, c2Head:%s",
			c1.Id(), c2.Id(), refer,
			c1.SprintTail(), c1.SprintHead(), c2.SprintTail(), c2.SprintHead()))
	} else if block.Hash() != tailHash {
		return errors.New(fmt.Sprintf("checkLink fail. c1:%s, c2:%s, refer:%t, c1Tail:%s, c1Head:%s, c2Tail:%s, c2Head:%s, hash[%s-%s]",
			c1.Id(), c2.Id(), refer,
			c1.SprintTail(), c1.SprintHead(), c2.SprintTail(), c2.SprintHead(), block.Hash(), tailHash))
	}
	return nil
}

func (self *blockPool) sprint(hash types.Hash) (commonBlock, *string) {
	b1, free := self.freeBlocks[hash]
	if free {
		return b1, nil
	}
	_, compound := self.compoundBlocks.Load(hash)
	if compound {
		s := "compound" + hash.String()
		return nil, &s
	}
	return nil, nil
}
func (self *blockPool) containsHash(hash types.Hash) bool {
	_, free := self.freeBlocks[hash]
	_, compound := self.compoundBlocks.Load(hash)
	return free || compound
}

func (self *blockPool) putBlock(hash types.Hash, pool commonBlock) {
	self.freeBlocks[hash] = pool
}

func (self *blockPool) compound(w commonBlock) {
	self.pendingMu.Lock()
	defer self.pendingMu.Unlock()
	self.compoundBlocks.Store(w.Hash(), true)
	delete(self.freeBlocks, w.Hash())
}

func (self *blockPool) delFromCompound(ws map[uint64]commonBlock) {
	self.pendingMu.Lock()
	defer self.pendingMu.Unlock()
	for _, b := range ws {
		self.compoundBlocks.Delete(b.Hash())
	}
}

func (self *blockPool) delHashFromCompound(hash types.Hash) {
	self.pendingMu.Lock()
	defer self.pendingMu.Unlock()
	self.compoundBlocks.Delete(hash)
}

func (self *BCPool) AddBlock(block commonBlock) {
	self.blockpool.pendingMu.Lock()
	defer self.blockpool.pendingMu.Unlock()
	hash := block.Hash()
	height := block.Height()
	if !self.blockpool.containsHash(hash) && !self.chainpool.tree.Exists(hash) {
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
	return self.blockpool.containsHash(hashes) || self.chainpool.tree.Exists(hashes)
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
			snippet := newSnippetChain(v, self.chainpool.genChainId())
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
	if len(self.chainpool.snippetChains) == 0 {
		return 0
	}
	i := 0
	sortSnippets := copyMap(self.chainpool.snippetChains)
	sort.Sort(ByTailHeight(sortSnippets))

	tmpChains := self.chainpool.allChain()

	for _, w := range sortSnippets {
		forky, insertable, c, err := self.chainpool.fork2(w, tmpChains)
		if err != nil {
			self.delSnippet(w)
			self.log.Error("fork to error.", "err", err)
			continue
		}
		if forky {
			i++
			newChain, err := self.chainpool.forkChain(c, w)
			if err == nil {
				self.delSnippet(w)
				// todo
				self.log.Info(fmt.Sprintf("insert new chain[%s][%s][%s],[%s][%s][%s]", c.Id(), c.SprintHead(), c.SprintTail(), newChain.Id(), newChain.SprintHead(), newChain.SprintTail()))
			}
			continue
		}
		if insertable {
			i++
			err := self.chainpool.insertSnippet(c, w)
			if err != nil {
				self.log.Error("insert fail.", "err", err)
			}
			self.delSnippet(w)
			continue
		}
	}
	dels := self.chainpool.tree.PruneTree()
	for _, c := range dels {
		self.log.Debug("del useless chain", "info", fmt.Sprintf("%+v", c.Id()), "tail", c.SprintTail(), "height", c.SprintHead())
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

	cur := self.CurrentChain()
	headH, _ := cur.HeadHH()
	head := new(big.Int).SetUint64(headH)

	//tailHeight, _ := cur.TailHH()

	i := 0
	zero := big.NewInt(0)
	prev := zero

	for _, w := range sortSnippets {
		// if snippet is lower, ignore
		//if w.headHeight+10 < tailHeight {
		//	continue
		//}
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
			diff.SetUint64(100)
		}

		i++
		hash := ledger.HashHeight{Hash: w.tailHash, Height: w.tailHeight}
		self.tools.fetcher.fetch(hash, diff.Uint64())

		prev.SetUint64(w.headHeight)
	}
	return i
}

func (self *BCPool) CurrentModifyToChain(target tree.Branch, hashH *ledger.HashHeight) error {
	return self.chainpool.tree.SwitchMainTo(target)
}

/**
If a block exists in chain and refer's chain at the same time, reduce the chain.
*/
//func reduceChainByRefer(target *forkedChain) []commonBlock {
//	var r []commonBlock
//	tailH := target.tailHeight
//	base := target.referChain
//
//	for i := tailH + 1; i <= target.headHeight; i++ {
//		b := target.getBlock(i, false)
//		baseB := base.getBlock(i, true)
//		if baseB != nil && baseB.Hash() == b.Hash() {
//			target.removeTail(b)
//			r = append(r, b)
//		}
//	}
//	return r
//}
func (self *BCPool) CurrentModifyToEmpty() error {
	return self.chainpool.tree.SwitchMainToEmpty()
}

func (self *BCPool) LongestChain() tree.Branch {
	readers := self.chainpool.allChain()
	current := self.CurrentChain()
	longest := current
	for _, reader := range readers {
		height, _ := reader.HeadHH()
		longestHeight, _ := longest.HeadHH()
		if height > longestHeight {
			longest = reader
		}
	}
	longestHeight, _ := longest.HeadHH()
	currentHeight, _ := current.HeadHH()
	if longestHeight-self.LIMIT_LONGEST_NUM > currentHeight {
		return longest
	} else {
		return current
	}
}
func (self *BCPool) LongerChain(minHeight uint64) []tree.Branch {
	var result []tree.Branch
	readers := self.chainpool.allChain()
	current := self.CurrentChain()
	for _, reader := range readers {
		if current.Id() == reader.Id() {
			continue
		}
		height, _ := reader.HeadHH()
		if height > minHeight {
			result = append(result, reader)
		}
	}
	return result
}
func (self *BCPool) CurrentChain() tree.Branch {
	return self.chainpool.tree.Main()
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
	//self.chainHeadMu.Lock()
	//defer self.chainHeadMu.Unlock()
	//
	//self.chainTailMu.Lock()
	//defer self.chainTailMu.Unlock()
	//
	//dels := make(map[string]*forkedChain)
	//height := self.chainpool.current.tailHeight
	//for _, c := range self.chainpool.allChain() {
	//	if c.headHeight+self.LIMIT_HEIGHT < height {
	//		dels[c.id()] = c
	//	}
	//}
	//for {
	//	i := 0
	//	for _, c := range self.chainpool.allChain() {
	//		_, ok := dels[c.id()]
	//		if ok {
	//			continue
	//		}
	//		r := c.refer()
	//		if r == nil {
	//			i++
	//			dels[c.id()] = c
	//		} else {
	//			_, ok := dels[r.id()]
	//			if ok {
	//				i++
	//				dels[c.id()] = c
	//			}
	//		}
	//	}
	//	if i == 0 {
	//		break
	//	} else {
	//		i = 0
	//	}
	//}
	//
	//for _, v := range dels {
	//	self.log.Info("del useless chain", "id", v.id(), "headHeight", v.headHeight, "tailHeight", v.tailHeight)
	//	self.delChain(v)
	//}
	//
	//for _, c := range self.chainpool.snippetChains {
	//	if c.headHeight+self.LIMIT_HEIGHT < height {
	//		self.delSnippet(c)
	//	}
	//}
}

//func (self *BCPool) delChain(c *forkedChain) {
//	self.chainpool.delChain(c.id())
//	self.blockpool.delFromCompound(c.heightBlocks)
//}
func (self *BCPool) delSnippet(c *snippetChain) {
	delete(self.chainpool.snippetChains, c.id())
	self.blockpool.delFromCompound(c.heightBlocks)
}
func (self *BCPool) info() map[string]interface{} {
	result := make(map[string]interface{})
	bp := self.blockpool
	cp := self.chainpool

	result["FreeSize"] = len(bp.freeBlocks)
	result["CompoundSize"] = common.SyncMapLen(&bp.compoundBlocks)
	result["SnippetSize"] = len(cp.snippetChains)
	result["TreeSize"] = cp.tree.Size()
	result["CurrentLen"] = cp.tree.Main().Size()
	var snippetIds []interface{}
	for _, v := range cp.snippetChains {
		snippetIds = append(snippetIds, v.info())
	}
	result["Snippets"] = snippetIds
	var chainIds []interface{}
	for _, v := range cp.allChain() {
		chainIds = append(chainIds, tree.PrintBranchInfo(v))
	}
	result["Chains"] = chainIds
	result["ChainSize"] = len(chainIds)
	result["Current"] = tree.PrintBranchInfo(cp.tree.Main())
	result["Disk"] = tree.PrintBranchInfo(cp.diskChain)

	return result
}

func (self *BCPool) detailChain(id string) map[string]interface{} {
	//c := self.chainpool.getChain(id)
	//if c != nil {
	//	return c.detail()
	//}
	//s := self.chainpool.snippetChains[id]
	//if s != nil {
	//	return s.detail()
	//}
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
