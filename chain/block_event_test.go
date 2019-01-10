package chain

import (
	"fmt"
	"math/big"
	"testing"
)

func TestChain_GetEvent(t *testing.T) {
	chainInstance := getChainInstance()
	latestBlockEventId, _ := chainInstance.GetLatestBlockEventId()
	for i := latestBlockEventId; i > 0; i-- {
		types, hashList, _ := chainInstance.GetEvent(i)
		if types == byte(4) || types == byte(2) {
			fmt.Printf("%d %v: %+v\n", i, types, hashList)
		}

	}

}

func TestDifficulty(t *testing.T) {
	new(big.Int).SetString("", 10)
}
