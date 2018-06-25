package main

import (
	"go-vite/ledger"
	"math/big"
	"fmt"
)

func main () {
	accountBlockTest()
	accountBlockChainTest()

	snapshotBlockTest()
	snapshotBlockChainTest()
}

func createAccountBlock () *ledger.AccountBlock {
	return &ledger.AccountBlock{
		Account: []byte{1, 2, 3},

		To: []byte{4, 5, 6},

		PrevHash: []byte{7, 8, 9},

		FromHash: []byte{10, 11, 12},

		BlockNum: big.NewInt(10),

		Balance: map[uint32]*big.Int {
			13: big.NewInt(14),

			15: big.NewInt(16),

			17: big.NewInt(18),
		},

		Signature: []byte{19, 20, 21},
	}
}

func createSnapshotBlock () *ledger.SnapshotBlock {
	return &ledger.SnapshotBlock{
		PrevHash: []byte{1, 2, 3},

		BlockNum: big.NewInt(4),

		Producer: []byte{5, 6, 7},

		Snapshot: map[string][]byte {
			"a": []byte{8, 9, 10},

			"b": []byte{11, 12, 13},

			"c": []byte{14, 15, 16},
		},

		Signature: []byte{19, 20, 21},
	}
}



func accountBlockChainTest () {
	accountBlockChain := ledger.AccountBlockChain{}.New()


	accountBlockChain.WriteBlock(createAccountBlock())

	accountBlock, _ := accountBlockChain.GetBlock([]byte("test"))
	fmt.Printf("%+v\n", accountBlock)
	fmt.Println()
}

func snapshotBlockChainTest () {
	snapshotBlockChain := ledger.SnapshotBlockChain{}.New()


	snapshotBlockChain.WriteBlock(createSnapshotBlock())

	snapshotBlock, _ := snapshotBlockChain.GetBlock([]byte("snapshot_test"))
	fmt.Printf("%+v\n", snapshotBlock)
	fmt.Println()
}


func accountBlockTest () {
	ab := createAccountBlock()

	result, _ := ab.Serialize()

	fmt.Printf("%+v\n", result)

	ab2 := &ledger.AccountBlock{}
	ab2.Deserialize(result)

	fmt.Printf("%+v\n", ab2)
	fmt.Println()
}

func snapshotBlockTest () {
	sb := createSnapshotBlock()

	result, _ := sb.Serialize()

	fmt.Printf("%+v\n", result)


	sb2 := &ledger.SnapshotBlock{}
	sb2.Deserialize(result)

	fmt.Printf("%+v\n", sb2)
	fmt.Println()
}
