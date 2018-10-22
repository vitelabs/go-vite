package pool

import (
	"testing"

	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

type mockForBcPool interface {
	chainRw
	commonSyncer
}

type testChainRw struct {
}

func (self *testChainRw) insertBlock(block commonBlock) error {
	panic("implement me")
}

func (self *testChainRw) insertBlocks(blocks []commonBlock) error {
	panic("implement me")
}

func (self *testChainRw) head() commonBlock {
	panic("implement me")
}

func (self *testChainRw) getBlock(height uint64) commonBlock {
	panic("implement me")
}

type testSyncer struct {
}

func (self *testSyncer) fetch(hashHeight ledger.HashHeight, prevCnt uint64) {
	panic("implement me")
}

func TestBcPool(t *testing.T) {
	bcPool := BCPool{}

	bcPool.Id = "test"
	bcPool.version = &ForkVersion{}
	bcPool.log = log15.New("module", "pool/test")

	testRw := &testChainRw{}
	bcPool.init(&tools{rw: testRw, fetcher: &testSyncer{}})
}
