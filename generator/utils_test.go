package generator

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"testing"
)

type mockChain struct {
	seed uint64
}

func (c mockChain) GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error) {
	return nil, nil
}
func (c mockChain) GetSnapshotBlockByContractMeta(addr *types.Address, fromHash *types.Hash) (*ledger.SnapshotBlock, error) {
	return nil, nil
}
func (c mockChain) GetSeed(limitSb *ledger.SnapshotBlock, fromHash types.Hash) (uint64, error) {
	return c.seed, nil
}

func TestSeed(t *testing.T) {
	seed := uint64(15637427697709333733)
	chain := mockChain{seed: seed}
	vmGlobalStatus := NewVMGlobalStatus(chain, nil, types.Hash{})
	for i := 0; i < 100; i++ {
		rand, err := vmGlobalStatus.Seed()
		if err != nil || rand != seed {
			t.Fatalf("before hard fork, expected [nil, %v], got [%v, %v]", seed, err, rand)
		}
	}

	seed = uint64(15637427697709333733)
	chain = mockChain{seed: seed}
	vmGlobalStatus = NewVMGlobalStatus(chain, nil, types.Hash{})
	randList := []uint64{3859229079807094838, 12152911693104629581, 9644037654042465595, 9243194358276174014, 11414992830392538450, 3925454220078744999, 16503802664012760397, 7980644548622968793, 13268626495317126268, 12067214392073868583}
	for index, expected := range randList {
		rand, err := vmGlobalStatus.Random()
		if err != nil || rand != expected {
			t.Fatalf("after hard fork, index %v, expected [nil, %v], got [%v, %v]", index, expected, err, rand)
		}
	}
}
