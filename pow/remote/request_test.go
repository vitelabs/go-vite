package remote

import (
	"flag"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/monitor"
	"math/big"
	"testing"
	"time"
)

func init() {
	flag.StringVar(&requestUrl, "url", "", "")
	flag.Parse()
}

func TestPowGenerate(t *testing.T) {
	defer monitor.LogTime("verify", "VerifyforVM", time.Now())
	addr, _, _ := types.CreateAddress()
	prevHash := types.ZERO_HASH
	difficulty := new(big.Int).SetUint64(0x000000000000FFFF)
	work, err := GenerateWork(types.DataListHash(addr.Bytes(), prevHash.Bytes()).Bytes(), difficulty.Uint64())
	if err != nil {
		t.Error(err.Error())
		return
	}
	fmt.Printf("calcData:%v\n", work)

	//var wg sync.WaitGroup
	//for i := 0; i < 5; i++ {
	//	wg.Add(1)
	//	go func() {
	//		defer wg.Done()
	//		lastTime := time.Now()
	//		for i := uint64(1); i <= 100; i++ {
	//			_, err := powRequest.GenerateWork(types.DataListHash(addr.Bytes(), prevHash.Bytes()), difficulty.Uint64())
	//			if err != nil {
	//				t.Error(err.Error())
	//				return
	//			}
	//			//fmt.Printf("calcData:%v\n", work)
	//		}
	//		endTime := time.Now()
	//		ts := uint64(endTime.Sub(lastTime).Nanoseconds())
	//		fmt.Printf("g: %d\n", ts/100)
	//	}()
	//}
	//wg.Wait()
}
