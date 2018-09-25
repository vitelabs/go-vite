package pool

import (
	"strconv"
	"testing"
	"time"

	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/test"
	"github.com/viteshan/naive-vite/verifier"
)

type TestVerifier struct {
}

type TestBlockVerifyStat struct {
	verifier.BlockVerifyStat
	result verifier.VerifyResult
}

func (self *TestBlockVerifyStat) VerifyResult() verifier.VerifyResult {
	return self.result
}

func (self *TestBlockVerifyStat) Reset() {
	self.result = verifier.PENDING
}

func (self *TestVerifier) VerifyReferred(block common.Block, stat verifier.BlockVerifyStat) {
	switch stat.(type) {
	case *TestBlockVerifyStat:
		testStat := stat.(*TestBlockVerifyStat)
		switch testStat.result {
		case verifier.PENDING:
			testStat.result = verifier.SUCCESS
		}
	}
}

func (self *TestVerifier) NewVerifyStat(t verifier.VerifyType, block common.Block) verifier.BlockVerifyStat {
	return &TestBlockVerifyStat{result: verifier.PENDING}
}

type TestSyncer struct {
	pool   *BCPool
	blocks map[string]*test.TestBlock
}

type TestChainReader struct {
	head  common.Block
	store map[int]common.Block
}

func (self *TestChainReader) GetBlock(height int) common.Block {
	return self.store[height]
}

func (self *TestChainReader) init() {
	self.store = make(map[int]common.Block)
	//self.head = genesis
	//self.store[genesis.Theight] = genesis
}

func (self *TestChainReader) Head() common.Block {
	return self.head
}
func (self *TestChainReader) insertChain(block common.Block, forkVersion int) error {
	log.Info("insert to forkedChain: %s", block)
	self.head = block
	//self.store[block.Height()] = block
	return nil
}

func (self *TestChainReader) removeChain(block common.Block) error {
	//log.Info("remove from forkedChain: %s", block)
	//self.head = self.store[block.Height()-1]
	//delete(self.store, block.Height())
	return nil
}
func (self *TestSyncer) fetch(hash common.HashHeight, prevCnt int) {
	//log.Info("fetch request,cnt:%d, hash:%v", prevCnt, hash)
	//go func() {
	//	prev := hash.Hash
	//
	//	for i := 0; i < prevCnt; i++ {
	//		block, ok := self.blocks[prev]
	//		if ok {
	//			log.Info("recv from net: %s", block)
	//			self.pool.AddBlock(block)
	//		} else {
	//			return
	//		}
	//		prev = block.PreHash()
	//	}
	//
	//}()
}
func (self *TestSyncer) FetchAccount(address string, hash common.HashHeight, prevCnt int) {
	self.fetch(hash, prevCnt)
}

func (self *TestSyncer) FetchSnapshot(hash common.HashHeight, prevCnt int) {
	self.fetch(hash, prevCnt)
}

func (self *TestSyncer) genLinkedData() {
	self.blocks = genLinkBlock("A-", 1, 100, genesis)
	block := self.blocks["A-5"]
	tmp := genLinkBlock("B-", 6, 30, block)
	for k, v := range tmp {
		self.blocks[k] = v
	}

	block = self.blocks["A-6"]
	tmp = genLinkBlock("C-", 7, 30, block)
	for k, v := range tmp {
		self.blocks[k] = v
	}
}

func genLinkBlock(mark string, start int, end int, genesis *test.TestBlock) map[string]*test.TestBlock {
	blocks := make(map[string]*test.TestBlock)
	last := genesis
	for i := start; i < end; i++ {
		hash := mark + strconv.Itoa(i)
		block := &test.TestBlock{Thash: hash, Theight: i, TpreHash: last.Hash(), Tsigner: signer}
		blocks[hash] = block
		last = block
	}
	return blocks
}

var genesis = &test.TestBlock{Thash: "A-0", Theight: 0, TpreHash: "-1", Tsigner: signer, Ttimestamp: time.Now()}

var signer = "viteshan"

func TestBcPool(t *testing.T) {

	//reader := &TestChainReader{head: genesis}
	//reader.init()
	//testSyncer := &TestSyncer{blocks: make(map[string]*test.TestBlock)}
	//testSyncer.genLinkedData()
	//pool := newBlockChainPool("bcPool-1")
	//testSyncer.pool = pool
	//
	//pool.init(&accountCh{address: "viteshan2", bc: chain2.NewChain()}, &TestVerifier{}, NewTools("", testSyncer))
	//go pool.loop()
	//pool.AddBlock(&test.TestBlock{Thash: "A-6", Theight: 6, TpreHash: "A-5", Tsigner: signer, Ttimestamp: time.Now()})
	//time.Sleep(time.Second)
	//pool.AddBlock(&test.TestBlock{Thash: "C-10", Theight: 10, TpreHash: "C-9", Tsigner: signer, Ttimestamp: time.Now()})
	//time.Sleep(time.Second)
	//pool.AddBlock(&test.TestBlock{Thash: "A-1", Theight: 1, TpreHash: "A-0", Tsigner: signer, Ttimestamp: time.Now()})
	//time.Sleep(time.Second)
	//
	//pool.AddBlock(&test.TestBlock{Thash: "A-20", Theight: 20, TpreHash: "A-19", Tsigner: signer, Ttimestamp: time.Now()})
	//pool.AddBlock(&test.TestBlock{Thash: "B-9", Theight: 9, TpreHash: "A-8", Tsigner: signer, Ttimestamp: time.Now()})
	//c := make(chan int)
	//c <- 1
}

func TestInsertChain(t *testing.T) {
	//reader := &TestChainReader{head: &test.TestBlock{Thash: "1", Theight: 1, TpreHash: "0", Tsigner: signer, Ttimestamp: time.Now()}}
	//reader.insertBlock(&test.TestBlock{Thash: "1", Theight: 1, TpreHash: "0", Tsigner: "viteshan", Ttimestamp: time.Now()}, 1)
}

func TestFetchSnippet(t *testing.T) {
	//head := 100
	//var sortSnippets []*snippetChain
	//
	//sortSnippets = append(sortSnippets, &snippetChain{chain: chain{tailHeight: 1004, headHeight: 1006}})
	//sortSnippets = append(sortSnippets, &snippetChain{chain: chain{tailHeight: 1005, headHeight: 1007}})
	//sort.Sort(ByTailHeight(sortSnippets))
	//i := 0
	//prev := -1
	//
	//for _, w := range sortSnippets {
	//	//println(w.tailHeight)
	//	diff := 0
	//	if prev > 0 {
	//		diff = w.tailHeight - prev
	//	} else {
	//		// first snippet
	//		diff = w.tailHeight - head
	//	}
	//
	//	// prev and this chains have fork
	//	if diff <= 0 {
	//		diff = w.tailHeight - head
	//	}
	//
	//	// lower than the current chain
	//	if diff <= 0 {
	//		diff = 20
	//	}
	//
	//	if diff > 0 {
	//		i++
	//		println(w.tailHeight, diff)
	//		//hash := common.HashHeight{Hash: w.tailHash, Height: w.tailHeight}
	//	}
	//	prev = w.headHeight
	//}
}
