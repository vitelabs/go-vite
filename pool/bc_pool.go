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

// BCPool is the basis for account pool and snapshot pool
type BCPool struct {
	ID  string
	log log15.Logger

	blockpool *blockPool
	chainpool *chainPool
	tools     *tools

	version *common.Version

	// 1. protecting the tail(hash && height) of current chain.
	// 2. protecting the modification for the current chain (which is the current chain?).
	chainTailMu sync.Mutex
	chainHeadMu sync.Mutex

	limitLongestNum uint64

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
	chainID      string
}

func (c *chain) size() uint64 {
	return c.headHeight - c.tailHeight
}
func (c *chain) HeadHeight() uint64 {
	return c.headHeight
}

func (c *chain) ChainID() string {
	return c.chainID
}

func (c *chain) id() string {
	return c.chainID
}

type snippetChain struct {
	chain
	tailHash types.Hash
	headHash types.Hash

	utime time.Time
}

func newSnippetChain(w commonBlock, id string) *snippetChain {
	self := &snippetChain{}
	self.chainID = id
	self.heightBlocks = make(map[uint64]commonBlock)
	self.headHeight = w.Height()
	self.headHash = w.Hash()
	self.tailHash = w.PrevHash()
	self.tailHeight = w.Height() - 1
	self.heightBlocks[w.Height()] = w
	self.utime = time.Now()
	return self
}

func (sc *snippetChain) addTail(w commonBlock) {
	sc.tailHash = w.PrevHash()
	sc.tailHeight = w.Height() - 1
	sc.heightBlocks[w.Height()] = w
	sc.utime = time.Now()
}
func (sc *snippetChain) deleteTail(newtail commonBlock) {
	sc.tailHash = newtail.Hash()
	sc.tailHeight = newtail.Height()
	delete(sc.heightBlocks, newtail.Height())
}
func (sc *snippetChain) remTail() commonBlock {
	newTail := sc.heightBlocks[sc.tailHeight+1]
	if newTail == nil {
		return nil
	}
	sc.deleteTail(newTail)
	return newTail
}

func (sc *snippetChain) merge(snippet *snippetChain) {
	sc.tailHeight = snippet.tailHeight
	sc.tailHash = snippet.tailHash
	for k, v := range snippet.heightBlocks {
		sc.heightBlocks[k] = v
	}
	sc.utime = time.Now()
}

func (sc *snippetChain) info() map[string]interface{} {
	result := make(map[string]interface{})
	result["TailHeight"] = sc.tailHeight
	result["TailHash"] = sc.tailHash
	result["HeadHeight"] = sc.headHeight
	result["HeadHash"] = sc.headHash
	result["Id"] = sc.id()
	return result
}
func (sc *snippetChain) getBlock(height uint64) commonBlock {
	block, ok := sc.heightBlocks[height]
	if ok {
		return block
	}
	return nil
}

func (bcp *BCPool) init(tools *tools) {
	bcp.tools = tools

	bcp.limitLongestNum = 3
	bcp.rstat = (&recoverStat{}).init(10, 10*time.Second)
	bcp.initPool()
}

func (bcp *BCPool) initPool() {
	diskChain := &branchChain{chainID: bcp.ID + "-diskchain", rw: bcp.tools.rw, v: bcp.version}

	t := tree.NewTree()
	chainpool := &chainPool{
		poolID:    bcp.ID,
		diskChain: diskChain,
		tree:      t,
		log:       bcp.log,
	}
	chainpool.init()
	blockpool := &blockPool{
		freeBlocks: make(map[types.Hash]commonBlock),
	}
	bcp.chainpool = chainpool
	bcp.blockpool = blockpool
}

func (bcp *BCPool) rollbackCurrent(blocks []commonBlock) error {
	if len(blocks) <= 0 {
		return nil
	}
	cur := bcp.CurrentChain()
	curTailHeight, curTailHash := cur.TailHH()
	// from small to big
	sort.Sort(ByHeight(blocks))

	bcp.log.Info("rollbackCurrent", "start", blocks[0].Height(), "end", blocks[len(blocks)-1].Height(), "size", len(blocks),
		"currentId", cur.ID())
	disk := bcp.chainpool.diskChain
	headHeight, headHash := disk.HeadHH()

	err := bcp.checkChain(blocks)
	if err != nil {
		bcp.log.Info("check chain fail." + bcp.printf(blocks))
		panic(err)
		return err
	}

	if cur.Linked(disk) {
		bcp.log.Info("poolChain and db is connected.", "headHeight", headHeight, "headHash", headHash)
		return nil
	}

	h := len(blocks) - 1
	smallest := blocks[0]
	longest := blocks[h]
	if headHeight+1 != smallest.Height() || headHash != smallest.PrevHash() {
		for _, v := range blocks {
			bcp.log.Info("block delete", "height", v.Height(), "hash", v.Hash(), "prevHash", v.PrevHash())
		}
		bcp.log.Crit("error for db fail.", "headHeight", headHeight, "headHash", headHash, "smallestHeight", smallest.Height(), "err", errors.New(bcp.ID+" disk chain height hash check fail"))
		return errors.New(bcp.ID + " disk chain height hash check fail")
	}

	if curTailHeight != longest.Height() || curTailHash != longest.Hash() {
		for _, v := range blocks {
			bcp.log.Info("block delete", "height", v.Height(), "hash", v.Hash(), "prevHash", v.PrevHash())
		}
		bcp.log.Crit("error for db fail.", "tailHeight", curTailHeight, "tailHash", curTailHash, "longestHeight", longest.Height(), "err", errors.New(bcp.ID+" current chain height hash check fail"))
		return errors.New(bcp.ID + " current chain height hash check fail")
	}
	main := bcp.chainpool.tree.Main()
	for i := h; i >= 0; i-- {
		err := bcp.chainpool.tree.AddTail(main, blocks[i])
		if err != nil {
			panic(err)
		}
	}

	err = tree.CheckTree(bcp.chainpool.tree)
	if err != nil {
		bcp.log.Error("rollbackCurrent check", "err", err)
	}
	return nil
}

// check blocks is a chain
func (bcp *BCPool) checkChain(blocks []commonBlock) error {
	var prev commonBlock
	for _, b := range blocks {
		if prev == nil {
			prev = b
			continue
		}
		if b.PrevHash() != prev.Hash() {
			return errors.New("not a chain")
		}
		if b.Height()-1 != prev.Height() {
			return errors.New("not a chain")
		}
		prev = b
	}
	return nil
}

// check blocks is a chain
func (bcp *BCPool) printf(blocks []commonBlock) string {
	result := ""
	for _, v := range blocks {
		result += fmt.Sprintf("[%d-%s-%s]", v.Height(), v.Hash(), v.PrevHash())
	}
	return result
}

func (bp *blockPool) sprint(hash types.Hash) (commonBlock, *string) {
	b1, free := bp.freeBlocks[hash]
	if free {
		return b1, nil
	}
	_, compound := bp.compoundBlocks.Load(hash)
	if compound {
		s := "compound" + hash.String()
		return nil, &s
	}
	return nil, nil
}
func (bp *blockPool) containsHash(hash types.Hash) bool {
	_, free := bp.freeBlocks[hash]
	_, compound := bp.compoundBlocks.Load(hash)
	return free || compound
}

func (bp *blockPool) putBlock(hash types.Hash, pool commonBlock) {
	bp.freeBlocks[hash] = pool
}

func (bp *blockPool) compound(w commonBlock) {
	bp.pendingMu.Lock()
	defer bp.pendingMu.Unlock()
	bp.compoundBlocks.Store(w.Hash(), true)
	delete(bp.freeBlocks, w.Hash())
}

func (bp *blockPool) delFromCompound(ws map[uint64]commonBlock) {
	bp.pendingMu.Lock()
	defer bp.pendingMu.Unlock()
	for _, b := range ws {
		bp.compoundBlocks.Delete(b.Hash())
	}
}

func (bp *blockPool) delHashFromCompound(hash types.Hash) {
	bp.pendingMu.Lock()
	defer bp.pendingMu.Unlock()
	bp.compoundBlocks.Delete(hash)
}

func (bcp *BCPool) addBlock(block commonBlock) {
	bcp.blockpool.pendingMu.Lock()
	defer bcp.blockpool.pendingMu.Unlock()
	hash := block.Hash()
	height := block.Height()
	if !bcp.blockpool.containsHash(hash) && !bcp.chainpool.tree.Exists(hash) {
		bcp.blockpool.putBlock(hash, block)
	} else {
		monitor.LogEvent("pool", "addDuplication")
		// todo online del
		bcp.log.Warn(fmt.Sprintf("block exists in BCPool. hash:[%s], height:[%d].", hash, height))
	}
}
func (bcp *BCPool) existInPool(hashes types.Hash) bool {
	bcp.blockpool.pendingMu.Lock()
	defer bcp.blockpool.pendingMu.Unlock()
	return bcp.blockpool.containsHash(hashes) || bcp.chainpool.tree.Exists(hashes)
}

// ByHeight sorts commonBlock by height
type ByHeight []commonBlock

func (a ByHeight) Len() int           { return len(a) }
func (a ByHeight) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByHeight) Less(i, j int) bool { return a[i].Height() < a[j].Height() }

// ByTailHeight sorts snippetChain by tail height
type ByTailHeight []*snippetChain

func (a ByTailHeight) Len() int           { return len(a) }
func (a ByTailHeight) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTailHeight) Less(i, j int) bool { return a[i].tailHeight < a[j].tailHeight }

func (bcp *BCPool) loopGenSnippetChains() int {
	if len(bcp.blockpool.freeBlocks) == 0 {
		return 0
	}

	i := 0
	// todo why copy ?
	sortPending := copyValuesFrom(bcp.blockpool.freeBlocks, &bcp.blockpool.pendingMu)
	sort.Sort(sort.Reverse(ByHeight(sortPending)))

	// todo why copy ?
	chains := copyMap(bcp.chainpool.snippetChains)

	for _, v := range sortPending {
		if !tryInsert(chains, v) {
			snippet := newSnippetChain(v, bcp.chainpool.genChainID())
			chains = append(chains, snippet)
		}
		i++
		bcp.blockpool.compound(v)
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
	bcp.chainpool.snippetChains = final
	return i
}

func (bcp *BCPool) loopAppendChains() int {
	if len(bcp.chainpool.snippetChains) == 0 {
		return 0
	}
	i := 0
	sortSnippets := copyMap(bcp.chainpool.snippetChains)
	sort.Sort(ByTailHeight(sortSnippets))

	tmpChains := bcp.chainpool.allChain()

	for _, w := range sortSnippets {
		forky, insertable, c, err := bcp.chainpool.fork2(w, tmpChains, bcp.blockpool)
		if err != nil {
			bcp.delSnippet(w)
			bcp.log.Error("fork to error.", "err", err)
			continue
		}
		if forky {
			i++
			newChain, err := bcp.chainpool.forkChain(c, w)
			if err == nil {
				bcp.delSnippet(w)
				// todo
				bcp.log.Info(fmt.Sprintf("insert new chain[%s][%s][%s],[%s][%s][%s]", c.ID(), c.SprintHead(), c.SprintTail(), newChain.ID(), newChain.SprintHead(), newChain.SprintTail()))
			}
			continue
		}
		if insertable {
			i++
			err := bcp.chainpool.insertSnippet(c, w)
			if err != nil {
				bcp.log.Error("insert fail.", "err", err)
			}
			bcp.delSnippet(w)
			continue
		}
	}

	return i
}
func (bcp *BCPool) loopFetchForSnippets() int {
	if len(bcp.chainpool.snippetChains) == 0 {
		return 0
	}
	sortSnippets := copyMap(bcp.chainpool.snippetChains)
	sort.Sort(ByTailHeight(sortSnippets))

	cur := bcp.CurrentChain()
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

		// don't fetch for new block
		b := w.getBlock(w.tailHeight + 1)
		if b != nil && !b.ShouldFetch() {
			continue
		}
		i++
		hash := ledger.HashHeight{Hash: w.tailHash, Height: w.tailHeight}
		bcp.tools.fetcher.fetch(hash, diff.Uint64())

		prev.SetUint64(w.headHeight)
	}
	return i
}

// CurrentModifyToChain switch target branch to main branch
func (bcp *BCPool) CurrentModifyToChain(target tree.Branch) error {
	t := bcp.chainpool.tree
	main := t.Main()
	bcp.log.Info("current modify", "targetId", target.ID(), "TargetTail", target.SprintTail(), "currentId", main.ID(), "CurrentTail", main.SprintTail())
	return t.SwitchMainTo(target)
}

// CurrentModifyToEmpty switch empty branch to main branch
func (bcp *BCPool) CurrentModifyToEmpty() error {
	return bcp.chainpool.tree.SwitchMainToEmpty()
}

// LongerChain looks for branches that are longer than the main branch
func (bcp *BCPool) LongerChain(minHeight uint64) []tree.Branch {
	var result []tree.Branch
	readers := bcp.chainpool.allChain()
	current := bcp.CurrentChain()
	for _, reader := range readers {
		if current.ID() == reader.ID() {
			continue
		}
		height, _ := reader.HeadHH()
		if height > minHeight {
			result = append(result, reader)
		}
	}
	return result
}

// CurrentChain returns the main branch
func (bcp *BCPool) CurrentChain() tree.Branch {
	return bcp.chainpool.tree.Main()
}

func (bcp *BCPool) loop() {
	for {
		bcp.loopGenSnippetChains()
		bcp.loopAppendChains()
		bcp.loopFetchForSnippets()
		time.Sleep(time.Second)
	}
}

func (bcp *BCPool) loopDelUselessChain() {
	bcp.chainHeadMu.Lock()
	defer bcp.chainHeadMu.Unlock()

	bcp.chainTailMu.Lock()
	defer bcp.chainTailMu.Unlock()

	for _, c := range bcp.chainpool.snippetChains {
		if c.utime.Add(time.Minute * 5).Before(time.Now()) {
			bcp.delSnippet(c)
			bcp.log.Info(fmt.Sprintf("delete snippet[%s][%d-%s][%d-%s]", c.id(), c.headHeight, c.headHash, c.tailHeight, c.tailHash))
		}
	}

	dels := bcp.chainpool.tree.PruneTree()
	for _, c := range dels {
		bcp.log.Debug("del useless chain", "info", fmt.Sprintf("%+v", c.ID()), "tail", c.SprintTail(), "height", c.SprintHead())
	}
}

func (bcp *BCPool) delSnippet(c *snippetChain) {
	delete(bcp.chainpool.snippetChains, c.id())
	bcp.blockpool.delFromCompound(c.heightBlocks)
}
func (bcp *BCPool) info() map[string]interface{} {
	result := make(map[string]interface{})
	bp := bcp.blockpool
	cp := bcp.chainpool

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

func (bcp *BCPool) detailChain(id string) map[string]interface{} {
	// todo
	return nil
}

func (bcp *BCPool) checkCurrent() {
	main := bcp.CurrentChain()
	tailHeight, tailHash := main.TailHH()
	headHeight, headHash := bcp.chainpool.diskChain.HeadHH()
	if headHeight != tailHeight || headHash != tailHash {
		panic(fmt.Sprintf("pool[%s] tail[%d-%s], chain head[%d-%s]",
			main.ID(), tailHeight, tailHash, headHeight, headHash))
	}
	err := tree.CheckTree(bcp.chainpool.tree)
	if err != nil {
		panic(err)
	}
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
