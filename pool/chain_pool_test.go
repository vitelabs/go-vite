package pool

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/pool/tree"
)

/**
                                                                  snippet
                                                                  +---+
                                                           +------+ 6'|
                                                           |      +---+
                                                           |
                                                           |
      +---+    +---+     +---+     +---+    +---+    +---+ |      +---+    +---+    +---+    +---+    +----+
disk  | 0 +----+ 1 +-----+ 2 +-----+ 3 +----+ 4 +----+ 5 | +------+ 6 +----+ 7 +----+ 8 +----+ 9 +----+ 10 |
      +---+    +---+     +---+     +---+    +---+    +---+        +---+    +---+    +---+    +---+    +----+
                                                                  main


*/
func Test_Forkable(t *testing.T) {
	tr := tree.NewTree()
	diskChain := tree.NewMockBranchRoot()
	{
		// init root
		flag := "root"
		for i := 0; i < 5; i++ {
			height, hash := diskChain.HeadHH()
			diskChain.AddHead(newMockCommonBlockByHH(height, hash, flag))
		}
		height, _ := diskChain.HeadHH()
		assert.Equal(t, height, uint64(5))
	}

	cp := &chainPool{
		poolID: "unittest",
		//diskChain: diskChain,
		tree: tr,
		log:  log15.New("module", "unittest"),
	}
	cp.snippetChains = make(map[string]*snippetChain)
	cp.tree.Init(cp.poolID, diskChain)

	main := tr.Main()
	height, _ := main.HeadHH()
	assert.Equal(t, height, uint64(5))
	height, _ = main.TailHH()
	assert.Equal(t, height, uint64(5))

	for i := 0; i < 6; i++ {
		height, hash := main.HeadHH()
		err := tr.AddHead(main, newMockCommonBlockByHH(height, hash, "main"))
		assert.Empty(t, err)
	}

	height, _ = main.HeadHH()
	assert.Equal(t, height, uint64(11))
	height, _ = main.TailHH()
	assert.Equal(t, height, uint64(5))

	knot := main.GetKnot(5, true)

	block := newMockCommonBlockByHH(knot.Height(), knot.Hash(), "snippet")
	snp := newSnippetChain(block, "snippet1")

	bs := tr.Branches()
	bm := make(map[string]tree.Branch)
	for _, v := range bs {
		bm[v.ID()] = v
	}
	forky, insertable, c, err := cp.fork2(snp, bm, nil)

	assert.Empty(t, err)
	assert.NotEmpty(t, c)
	assert.Equal(t, c.ID(), main.ID())
	assert.False(t, insertable)
	assert.True(t, forky)
}

/**

         snippet
          +---+
          | 2 |
          +-+-+
            |
+---+       |
| 1 | +-----+    main is empty
+---+
disk


*/
func Test_Forkable2(t *testing.T) {
	tr := tree.NewTree()
	diskChain := tree.NewMockBranchRoot()
	{
		height, hash := diskChain.HeadHH()
		diskChain.AddHead(newMockCommonBlockByHH(height, hash, "root"))
		height, hash = diskChain.HeadHH()
		assert.Equal(t, height, uint64(1))
	}

	cp := &chainPool{
		poolID: "unittest",
		//diskChain: diskChain,
		tree: tr,
		log:  log15.New("module", "unittest"),
	}
	cp.snippetChains = make(map[string]*snippetChain)
	cp.tree.Init(cp.poolID, diskChain)

	main := tr.Main()
	height, hash := main.HeadHH()
	assert.Equal(t, height, uint64(1))
	height, _ = main.TailHH()
	assert.Equal(t, height, uint64(1))

	block := newMockCommonBlockByHH(1, hash, "snippet")
	snp := newSnippetChain(block, "snippet1")

	bs := tr.Branches()
	bm := make(map[string]tree.Branch)
	for _, v := range bs {
		bm[v.ID()] = v
	}
	forky, insertable, c, err := cp.fork2(snp, bm, nil)

	assert.Empty(t, err)
	assert.NotEmpty(t, c)
	assert.Equal(t, c.ID(), main.ID())
	assert.True(t, insertable)
	assert.False(t, forky)
}
