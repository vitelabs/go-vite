package pool

import (
	"testing"

	"fmt"

	"encoding/json"

	"github.com/vitelabs/go-vite/common/types"
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

func Test_getForkPoint(t *testing.T) {
	base := mockChain(nil, 1, 1, 10)

	printChain(base)
	current := mockChain(base, 1, 10, 20)
	current.referChain = base

	printChain(current)
	longest := mockChain(current, 2, 15, 21)

	printChain(longest)

	bcPool := BCPool{}
	key, forked, e := bcPool.getForkPoint(longest, current)
	t.Log(key.Height(), forked.Height(), e)
}

func TestForkedChain_GetBlock(t *testing.T) {
	base := mockChain(nil, 1, 1, 10)
	current := mockChain(base, 2, 10, 20)
	c3 := mockChain(current, 3, 20, 30)
	printChain(c3)
	c4 := mockChain(c3, 4, 30, 40)
	current.referChain = c4
	printChain(c4)
}

func mockBlocks(sign int, start uint64, end uint64) []commonBlock {
	var result []commonBlock
	for i := start; i < end; i++ {
		result = append(result, newMockCommonBlock(sign, i))
	}
	return result
}

func mockChain(base *forkedChain, sign int, start uint64, head uint64) *forkedChain {
	var result []*mockCommonBlock
	for i := start; i <= head; i++ {
		result = append(result, newMockCommonBlock(sign, i))
	}
	c := &forkedChain{}
	if base != nil {
		b := base.GetBlock(start - 1)
		result[0].prevHash = b.Hash()
	}
	c.init(result[0])
	c.tailHash = result[0].prevHash
	c.tailHeight = result[0].height - 1
	c.headHash = result[0].prevHash
	c.headHeight = result[0].height - 1
	c.referChain = base
	for _, v := range result {
		c.addHead(v)
	}

	return c
}

func printChain(base *forkedChain) {
	byt, _ := json.Marshal(base.info())
	fmt.Println("-------------start--------------" + string(byt))
	for i := uint64(1); i <= base.headHeight; i++ {
		b := base.GetBlock(i)
		if b == nil {
			fmt.Println(i, "nil", "nil")
		} else {
			fmt.Println(b.Height(), b.Hash(), b.PrevHash())
		}
	}
	fmt.Println("-------------end--------------")
}

func printChainJust(base *forkedChain) {
	byt, _ := json.Marshal(base.info())
	fmt.Println("-------------start--------------" + string(byt))
	for i := uint64(base.tailHeight + 1); i <= base.headHeight; i++ {
		b := base.getHeightBlock(i)
		if b == nil {
			fmt.Println(i, "nil", "nil")
		} else {
			fmt.Println(b.Height(), b.Hash(), b.PrevHash())
		}
	}
	fmt.Println("-------------end--------------")
}

func TestGid(t *testing.T) {
	println(types.SNAPSHOT_GID.String())
}
