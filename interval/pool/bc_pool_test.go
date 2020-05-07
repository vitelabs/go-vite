package pool

import (
	"strconv"
	"testing"
	"time"

	"sort"

	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/test"
	"github.com/vitelabs/go-vite/interval/verifier"
)

type TestVerifier struct {
}

type TestBlockVerifyStat struct {
	verifier.BlockVerifyStat
	result verifier.VerifyResult
}

func (tstat *TestBlockVerifyStat) VerifyResult() verifier.VerifyResult {
	return tstat.result
}

func (tstat *TestBlockVerifyStat) Reset() {
	tstat.result = verifier.PENDING
}

func (v *TestVerifier) VerifyReferred(block common.Block, stat verifier.BlockVerifyStat) {
	switch stat.(type) {
	case *TestBlockVerifyStat:
		testStat := stat.(*TestBlockVerifyStat)
		switch testStat.result {
		case verifier.PENDING:
			testStat.result = verifier.SUCCESS
		}
	}
}

func (v *TestVerifier) NewVerifyStat(t verifier.VerifyType, block common.Block) verifier.BlockVerifyStat {
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

func (chReader *TestChainReader) GetBlock(height int) common.Block {
	return chReader.store[height]
}

func (chReader *TestChainReader) init() {
	chReader.store = make(map[int]common.Block)
	chReader.head = genesis
	chReader.store[genesis.Theight] = genesis
}

func (chReader *TestChainReader) Head() common.Block {
	return chReader.head
}
func (chReader *TestChainReader) insertChain(block common.Block, forkVersion int) error {
	log.Info("insert to forkedChain: %s", block)
	chReader.head = block
	chReader.store[block.Height()] = block
	return nil
}

func (chReader *TestChainReader) removeChain(block common.Block) error {
	log.Info("remove from forkedChain: %s", block)
	chReader.head = chReader.store[block.Height()-1]
	delete(chReader.store, block.Height())
	return nil
}
func (syn *TestSyncer) fetch(hash common.HashHeight, prevCnt int) {
	log.Info("fetch request,cnt:%d, hash:%v", prevCnt, hash)
	go func() {
		prev := hash.Hash

		for i := 0; i < prevCnt; i++ {
			block, ok := syn.blocks[prev]
			if ok {
				log.Info("recv from net: %s", block)
				syn.pool.AddBlock(block)
			} else {
				return
			}
			prev = block.PreHash()
		}

	}()
}
func (syn *TestSyncer) FetchAccount(address string, hash common.HashHeight, prevCnt int) {
	syn.fetch(hash, prevCnt)
}

func (syn *TestSyncer) FetchSnapshot(hash common.HashHeight, prevCnt int) {
	syn.fetch(hash, prevCnt)
}

func (syn *TestSyncer) genLinkedData() {
	syn.blocks = genLinkBlock("A-", 1, 100, genesis)
	block := syn.blocks["A-5"]
	tmp := genLinkBlock("B-", 6, 30, block)
	for k, v := range tmp {
		syn.blocks[k] = v
	}

	block = syn.blocks["A-6"]
	tmp = genLinkBlock("C-", 7, 30, block)
	for k, v := range tmp {
		syn.blocks[k] = v
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
	//pool.init(&accountCh{address: "viteshan2", bc: chain2.NewChain()}, &TestVerifier{}, NewFetcher("", testSyncer))
	//go pool.loop()
	//pool.AddBlock(&test.TestBlock{Thash: "A-6", Theight: 6, TprevHash: "A-5", Tsigner: signer, Ttimestamp: time.Now()})
	//time.Sleep(time.Second)
	//pool.AddBlock(&test.TestBlock{Thash: "C-10", Theight: 10, TprevHash: "C-9", Tsigner: signer, Ttimestamp: time.Now()})
	//time.Sleep(time.Second)
	//pool.AddBlock(&test.TestBlock{Thash: "A-1", Theight: 1, TprevHash: "A-0", Tsigner: signer, Ttimestamp: time.Now()})
	//time.Sleep(time.Second)
	//
	//pool.AddBlock(&test.TestBlock{Thash: "A-20", Theight: 20, TprevHash: "A-19", Tsigner: signer, Ttimestamp: time.Now()})
	//pool.AddBlock(&test.TestBlock{Thash: "B-9", Theight: 9, TprevHash: "A-8", Tsigner: signer, Ttimestamp: time.Now()})
	//c := make(chan int)
	//c <- 1
}

func TestInsertChain(t *testing.T) {
	reader := &TestChainReader{head: &test.TestBlock{Thash: "1", Theight: 1, TpreHash: "0", Tsigner: signer, Ttimestamp: time.Now()}}
	reader.insertChain(&test.TestBlock{Thash: "1", Theight: 1, TpreHash: "0", Tsigner: "viteshan", Ttimestamp: time.Now()}, 1)
}

func TestFetchSnippet(t *testing.T) {

	head := 100
	var sortSnippets []*snippetChain

	sortSnippets = append(sortSnippets, &snippetChain{chain: chain{tailHeight: 1004, headHeight: 1006}})
	sortSnippets = append(sortSnippets, &snippetChain{chain: chain{tailHeight: 1005, headHeight: 1007}})
	sort.Sort(ByTailHeight(sortSnippets))
	i := 0
	prev := -1

	for _, w := range sortSnippets {
		//println(w.tailHeight)
		diff := 0
		if prev > 0 {
			diff = w.tailHeight - prev
		} else {
			// first snippet
			diff = w.tailHeight - head
		}

		// prev and this chains have fork
		if diff <= 0 {
			diff = w.tailHeight - head
		}

		// lower than the current chain
		if diff <= 0 {
			diff = 20
		}

		if diff > 0 {
			i++
			println(w.tailHeight, diff)
			//hash := common.HashHeight{Hash: w.tailHash, Height: w.tailHeight}
		}
		prev = w.headHeight
	}
}
