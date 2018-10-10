package pool

import (
	"testing"

	ch "github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/wallet"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/consensus"
)

func TestNewPool(t *testing.T) {
	c := ch.NewChain(&config.Config{
		DataDir: common.DefaultDataDir(),
	})
	p := NewPool(c)
	w := wallet.New(nil)

	av := verifier.NewAccountVerifier(c, nil, w.KeystoreManager)

	cs := &consensus.MockConsensus{}
	sv := verifier.NewSnapshotVerifier(c, cs)

	c.Init()
	c.Start()
	p.Init(&MockSyncer{}, w, sv, av)

	p.Start()

	block := c.GetLatestSnapshotBlock()
	t.Log(block.Height, block.Hash, block.PrevHash, block.Producer())
	for k, v := range block.SnapshotContent {
		t.Log(k.String(), v.Hash, v.Height)
	}
}
