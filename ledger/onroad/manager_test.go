package onroad

import (
	"testing"
	"time"

	"github.com/vitelabs/go-vite/v2/common/config"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/ledger/test_tools"
)

func generateUnlockAddress() types.Address {
	return types.Address{}
}

func TestManager_ContractWorker(t *testing.T) {
	c, tempDir := test_tools.NewTestChainInstance(t.Name(), true, config.MockGenesis())
	defer test_tools.ClearChain(c, tempDir)
	
	v := NewMockVite(c)

	addr := generateUnlockAddress()
	v.Producer().(*mockProducer).Addr = addr

	manager := NewManager(v.Net(), v.Pool(), v.Producer(), nil, nil)
	manager.Init(v.chain)
	manager.Start()

	time.AfterFunc(1*time.Second, func() {
		t.Log("test c produceEvent ")
		v.Producer().(*mockProducer).produceEvent(1 * time.Second)
	})

	time.Sleep(3 * time.Second)
	manager.Stop()
}
