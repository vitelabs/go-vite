package ledger

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"testing"
)

func TestVmLogList_Hash(t *testing.T) {
	var vmLogList VmLogList

	fork.SetForkPoints(&config.ForkPoints{
		SeedFork: &config.ForkPoint{
			Height:  90,
			Version: 1,
		},
	})

	hash1, err1 := types.HexToHash("0dede580455f970517210ae2b9c0fbba74d5b7eea07eb0c62725e06c45061711")
	if err1 != nil {
		t.Fatal(err1)
	}

	hash2, err2 := types.HexToHash("c512660e3ee7d1dc005a6206ccaf84e6b567592d543f537566d871c2440e187e")
	if err2 != nil {
		t.Fatal(err2)
	}

	vmLogList = append(vmLogList, &VmLog{
		Topics: []types.Hash{hash1, hash2},
		Data:   []byte("test"),
	})

	address, err := types.HexToAddress("vite_544faefbc5031341c9352b0b22161bc0b4e5b342dc7fe04028")
	if err != nil {
		t.Fatal(err)
	}
	prehash, err := types.HexToHash("b7a12797d132c6545b5cc15afa8bb3811ac655f7a56cf76bad68bead4dfe5a44")
	if err != nil {
		t.Fatal(err)
	}
	vmLogHash1 := vmLogList.Hash(1, address, prehash)
	vmLogHash50 := vmLogList.Hash(50, address, prehash)
	vmLogHash100 := vmLogList.Hash(100, address, prehash)

	if *vmLogHash1 != *vmLogHash50 {
		t.Fatal(fmt.Sprintf("vmloghash1 should be equal with vmloghash50 , %+v, %+v", vmLogHash1, vmLogHash50))
	}

	if *vmLogHash100 == *vmLogHash1 {
		t.Fatal(fmt.Sprintf("vmloghash1 should not be equal with vmloghash100 , %+v, %+v", vmLogHash100, vmLogHash1))
	}
	fork.SetForkPoints(&config.ForkPoints{
		SeedFork: &config.ForkPoint{
			Height:  101,
			Version: 1,
		},
	})

	vmLogHash95 := vmLogList.Hash(95, address, prehash)

	if *vmLogHash50 != *vmLogHash95 {
		t.Fatal(fmt.Sprintf("vmloghash51 should be equal with vmloghash50 , %+v, %+v", vmLogHash95, vmLogHash50))
	}

	vmLogHash105 := vmLogList.Hash(105, address, prehash)
	if *vmLogHash105 != *vmLogHash100 {
		t.Fatal(fmt.Sprintf("vmloghash105 should be equal with vmLogHash100 , %+v, %+v", vmLogHash105, vmLogHash100))
	}

}
