package remote

import (
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/pow"
	"math/big"
	"testing"
	"time"
)

func init() {
	flag.StringVar(&requestUrl, "url", "", "")
	flag.Parse()
}

func TestPowGenerate(t *testing.T) {
	defer monitor.LogTime("pow", "remote", time.Now())
	InitRawUrl("http://127.0.0.1:6007")
	addr, _, _ := types.CreateAddress()
	prevHash := types.ZERO_HASH
	//difficulty := "FFFFFFC000000000000000000000000000000000000000000000000000000000"
	//realDifficulty, ok := new(big.Int).SetString(difficulty, 10)
	//if !ok {
	//	t.Error("string to big.Int failed")
	//}
	difficulty := big.NewInt(201564160)
	work, err := GenerateWork(types.DataListHash(addr.Bytes(), prevHash.Bytes()).Bytes(), difficulty)
	if err != nil {
		t.Error(err.Error())
		return
	}
	fmt.Printf("calcData:%v\n", work)

	nonceBig, ok := new(big.Int).SetString(*work, 16)
	if !ok {
		t.Error("wrong nonce str")
		return
	}
	nonceUint64 := nonceBig.Uint64()
	nn := make([]byte, 8)
	binary.LittleEndian.PutUint64(nn[:], nonceUint64)

	if !pow.CheckPowNonce(difficulty, nn, types.DataListHash(addr.Bytes(), prevHash.Bytes()).Bytes()) {
		t.Error("check nonce failed")
		return
	}

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
