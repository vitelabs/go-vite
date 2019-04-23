package tree

import (
	"encoding/json"
	"testing"

	"gotest.tools/assert"
)

func TestTree_SwitchMainTo(t *testing.T) {
	root := newMockBranchRoot()
	tr := NewTree()

	{
		// init root
		flag := "root"
		for i := 0; i < 5; i++ {
			root.addHead(newMockKnot(root.Head(), flag))
		}
		assert.Equal(t, root.headHeight, uint64(5))
	}

	tr.Init("unittest", root)

	var b3 Branch

	{
		flag := "b2"
		main := tr.Main()
		height, hash := main.HeadHH()
		b2 := tr.ForkBranch(main, height, hash)
		for i := 0; i < 3; i++ {
			h1, h2 := b2.HeadHH()
			tr.AddHead(b2, newMockKnotByHH(h1, h2, flag))
		}
	}

	{
		// add main and root
		flag := "root"
		for i := 0; i < 8; i++ {
			k := newMockKnot(root.Head(), flag)
			root.addHead(k)
			tr.RootHeadAdd(k)
		}
	}
	{
		flag := "b3"
		main := tr.Main()
		height, hash := main.HeadHH()
		b3 = tr.ForkBranch(main, height, hash)
		for i := 0; i < 3; i++ {
			h1, h2 := b3.HeadHH()
			tr.AddHead(b3, newMockKnotByHH(h1, h2, flag))
		}
	}
	{
		flag := "b1"
		main := tr.Main()
		for i := 0; i < 3; i++ {
			h1, h2 := main.HeadHH()
			tr.AddHead(main, newMockKnotByHH(h1, h2, flag))
		}
	}

	{
		// print tree
		msg := PrintTree(tr)
		byt, _ := json.Marshal(msg)
		t.Log(string(byt))
	}

	err := tr.SwitchMainTo(b3)
	assert.NilError(t, err)

	{
		// print tree
		msg := PrintTree(tr)
		byt, _ := json.Marshal(msg)
		t.Log(string(byt))
	}

	err = tr.SwitchMainToEmpty()
	assert.NilError(t, err)

	{ // print tree
		msg := PrintTree(tr)
		byt, _ := json.Marshal(msg)
		t.Log(string(byt))
	}

	err = CheckTree(tr)
	assert.NilError(t, err)

	err = CheckTreeSize(tr)
	assert.NilError(t, err)
}
