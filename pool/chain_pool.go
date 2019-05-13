package pool

import (
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/vitelabs/go-vite/pool/tree"

	"sync"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
)

type chainPool struct {
	poolID          string
	log             log15.Logger
	lastestChainIdx int32
	//current         *forkedChain
	snippetChains map[string]*snippetChain // head is fixed

	tree tree.Tree
	//chains          map[string]*forkedChain
	diskChain *branchChain

	chainMu sync.Mutex
}

func (cp *chainPool) forkChain(forked tree.Branch, snippet *snippetChain) (tree.Branch, error) {
	new := cp.tree.ForkBranch(forked, snippet.tailHeight, snippet.tailHash)
	for i := snippet.tailHeight + 1; i <= snippet.headHeight; i++ {
		v := snippet.heightBlocks[i]
		err := cp.tree.AddHead(new, v)
		if err != nil {
			return nil, err
		}
	}
	return new, nil
}

func (cp *chainPool) forkFrom(forked tree.Branch, height uint64, hash types.Hash) (tree.Branch, error) {
	forkedHeight, forkedHash := forked.HeadHH()
	if forkedHash == hash && forkedHeight == height {
		return forked, nil
	}

	new := cp.tree.ForkBranch(forked, height, hash)
	return new, nil
}

func (cp *chainPool) genChainID() string {
	return cp.poolID + "-" + strconv.Itoa(cp.incChainIdx())
}

func (cp *chainPool) incChainIdx() int {
	for {
		old := cp.lastestChainIdx
		new := old + 1
		if atomic.CompareAndSwapInt32(&cp.lastestChainIdx, old, new) {
			return int(new)
		}
	}
}
func (cp *chainPool) init() {
	cp.snippetChains = make(map[string]*snippetChain)
	cp.tree.Init(cp.poolID, cp.diskChain)
}

func (cp *chainPool) fork2(snippet *snippetChain, chains map[string]tree.Branch, bp *blockPool) (bool, bool, tree.Branch, error) {

	var forky, insertable bool
	var result tree.Branch
	var hr tree.Branch

	var err error

	trace := ""

LOOP:
	for _, c := range chains {
		tH := snippet.tailHeight
		tHash := snippet.tailHash
		block, reader := c.GetKnotAndBranch(tH)
		if block == nil || block.Hash() != tHash {
			continue
		}

		for i := tH + 1; i <= snippet.headHeight; i++ {
			trace = ""
			b2, r2 := c.GetKnotAndBranch(i)
			sb := snippet.getBlock(i)
			if b2 == nil {
				forky = false
				insertable = true
				hr = reader
				trace += "[1]"
				break LOOP
			}
			if b2.Hash() != sb.Hash() {
				if r2.ID() == reader.ID() {
					forky = true
					insertable = false
					hr = reader
					trace += "[2]"
					break LOOP
				}

				rHeight, rHash := reader.HeadHH()
				if rHeight == tH && rHash == tHash {
					forky = false
					insertable = true
					hr = reader
					trace += "[3]"
					break LOOP
				}
				break
			} else {
				reader = r2
				block = b2
				// todo
				cp.log.Info(fmt.Sprintf("block[%s-%d] exists. del from tail.", sb.Hash(), sb.Height()))
				tail := snippet.remTail()
				if tail == nil {
					delete(cp.snippetChains, snippet.id())
					hr = nil
					trace += "[4]"
					err = errors.Errorf("snippet rem nil. size:%d", snippet.size())
					break LOOP
				}
				bp.delHashFromCompound(tail.Hash())
				if snippet.size() == 0 {
					delete(cp.snippetChains, snippet.id())
					hr = nil
					trace += "[5]"
					err = errors.New("snippet is empty")
					break LOOP
				}
				tH = tail.Height()
				tHash = tail.Hash()
			}
		}
	}

	if err != nil {
		return false, false, nil, err
	}

	if hr == nil {
		return false, false, nil, nil
	}
	switch hr.Type() {
	case tree.Disk:
		result = cp.tree.Main()
		if insertable {
			mHeight, mHash := cp.tree.Main().HeadHH()
			if mHeight == snippet.tailHeight && mHash == snippet.tailHash {
				forky = false
				insertable = true
				result = cp.tree.Main()
				trace += "[5]"
			} else {
				forky = true
				insertable = false
				result = cp.tree.Main()
				trace += "[6]"
			}
		}
	case tree.Normal:
		trace += "[7]"
		result = hr
	}
	// todo
	//if insertable {
	//	err := result.canAddHead(snippet.getBlock(snippet.tailHeight + 1))
	//	if err != nil {
	//		cp.log.Error("fork2 fail.",
	//			"sTailHeight", snippet.tailHeight, "sHeadHeight",
	//			snippet.headHeight, "cTailHeight", result.tailHeight, "cHeadHeight", result.headHeight,
	//			"trace", trace)
	//		return forky, false, result, nil
	//	}
	//}

	return forky, insertable, result, nil
}

func (cp *chainPool) insertSnippet(c tree.Branch, snippet *snippetChain) error {
	for i := snippet.tailHeight + 1; i <= snippet.headHeight; i++ {
		w := snippet.heightBlocks[i]
		err := cp.insert(c, w)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cp *chainPool) insert(c tree.Branch, wrapper commonBlock) error {
	if cp.tree.Main().ID() == c.ID() {
		// todo remove
		cp.log.Info(fmt.Sprintf("insert to current[%s]:[%s-%d]%s", c.ID(), wrapper.Hash(), wrapper.Height(), wrapper.Latency()))
	} else {
		cp.log.Info(fmt.Sprintf("insert to chain[%s]:[%s-%d]%s", c.ID(), wrapper.Hash(), wrapper.Height(), wrapper.Latency()))
	}
	height, hash := c.HeadHH()
	if wrapper.Height() == height+1 {
		if hash == wrapper.PrevHash() {
			err := cp.tree.AddHead(c, wrapper)
			if err != nil {
				panic(err)
			}
			return nil
		}
		return errors.Errorf("forkedChain fork, fork point height[%d],hash[%s], but next block[%s]'s preHash is [%s]",
			height, hash, wrapper.Hash(), wrapper.PrevHash())
	}
	return errors.Errorf("forkedChain fork, fork point height[%d],hash[%s], but next block[%s]'s preHash[%s]-[%d]",
		height, hash, wrapper.Hash(), wrapper.PrevHash(), wrapper.Height())
}

func (cp *chainPool) insertNotify(head commonBlock) {
	err := cp.tree.RootHeadAdd(head)
	if err != nil {
		panic(err)
	}
}

func (cp *chainPool) writeBlockToChain(block commonBlock) error {
	err := cp.diskChain.rw.insertBlock(block)

	if err != nil {
		// todo opt log
		cp.log.Error(fmt.Sprintf("pool insert Chain fail. height:[%d], hash:[%s]", block.Height(), block.Hash()))
		return err
	}
	return cp.tree.RootHeadAdd(block)
}

func (cp *chainPool) allChain() map[string]tree.Branch {
	return cp.tree.Branches()
}
