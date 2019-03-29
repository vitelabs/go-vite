package consensus

import (
	"fmt"
	"testing"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/consensus/db"
)

func TestPut(t *testing.T) {
	cache := &DbCache{db: consensus_db.NewConsensusDBByDir("/Users/jie/Documents/vite/src/github.com/vitelabs/cache")}

	hashes, err := types.HexToHash("44b7847bd510f63ea2d8a23e618d7f3b6ca69775d367a56cd5d341e6cfe04ad6")
	if err != nil {
		panic(err)
	}
	//
	//var addrArr []types.Address
	//addr, err := types.HexToAddress("vite_29a3ec96d1e4a52f50e3119ed9945de09bef1d74a772ee60ff")
	//if err != nil {
	//	panic(err)
	//}
	//addrArr = append(addrArr, addr)
	//
	//addr, err = types.HexToAddress("vite_0d97a04d61dbbf238e9b6050c9d29d2b6f97986859e7856ec9")
	//if err != nil {
	//	panic(err)
	//}
	//addrArr = append(addrArr, addr)
	//cache.Add(hashes, addrArr)
	result, ok := cache.Get(hashes)
	fmt.Println(result, ok)
}
