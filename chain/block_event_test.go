package chain

import (
	"fmt"
	"testing"
)

func TestChain_GetEvent(t *testing.T) {
	chainInstance := getChainInstance()
	latestBlockEventId, _ := chainInstance.GetLatestBlockEventId()
	for i := latestBlockEventId; i > 0; i-- {
		types, hashList, _ := chainInstance.GetEvent(i)
		if types == 4 {
			fmt.Printf("%d %v: %+v\n", i, types, hashList)
		}
	}

}
