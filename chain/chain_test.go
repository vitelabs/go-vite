package chain

import (
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/config"
	"testing"
)

func TestNewChain(t *testing.T) {
	c := NewChain(&config.Config{
		DataDir: common.DefaultDataDir(),
	})
	c.Init()
	c.Start()

}
