package consensus

import (
	"os"
	"testing"

	"github.com/vitelabs/go-vite/log15"

	"github.com/vitelabs/go-vite/chain"
)

func testDataDir() string {
	return "testdata-consensus"
}

func prepareChain() chain.Chain {
	clearChain(nil)
	c := chain.NewChain(testDataDir())
	err := c.Init()
	if err != nil {
		panic(err)
	}
	err = c.Start()
	if err != nil {
		panic(err)
	}
	return c
}

func clearChain(c chain.Chain) {
	if c != nil {
		c.Stop()
	}
	err := os.RemoveAll(testDataDir())
	if err != nil {
		panic(err)
	}
}

func Test_chainRw(t *testing.T) {
	c := prepareChain()
	defer clearChain(c)

	log := log15.New("unittest", "chainrw")
	rw := newChainRw(c, log)
	rw.initArray(nil)
}
