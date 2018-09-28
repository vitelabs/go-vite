package chain

import (
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/config"
	"os"
	"testing"
)

var innerChainInstance Chain

func getChainInstance() Chain {
	if innerChainInstance == nil {
		innerChainInstance = NewChain(&config.Config{
			DataDir: common.DefaultDataDir(),
		})
		innerChainInstance.Init()
		innerChainInstance.Start()
	}
	return innerChainInstance
}

func TestMain(m *testing.M) {

	os.Exit(m.Run())
}
