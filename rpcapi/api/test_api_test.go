package api

import (
	"context"
	"fmt"
	"github.com/hashicorp/golang-lru"
	"github.com/robfig/cron"
	"github.com/vitelabs/go-vite/common/math"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

func TestTestApi_GetTestToken(t *testing.T) {
	a := rand.Int() % 1000
	a += 1
	ba := new(big.Int).SetInt64(int64(a))
	ba.Mul(ba, math.BigPow(10, 18))

	amount := ba.String()
	fmt.Println(amount)
}

func TestTestApi_CheckFrequent(t *testing.T) {
	cache, _ := lru.New(3)

	mc := cron.New()
	mc.AddFunc("10 * * * * *", func() {
		fmt.Println("clear lru ", time.Now())
		cache, _ = lru.New(3)
	})
	mc.Start()
	go func() {
		for i := 0; i < 30; i++ {
			if i == 22 {
				fmt.Println("sleep ", time.Now())
				time.Sleep(60 * time.Second)
				fmt.Println("wake ", time.Now())
			}
			{
				ctx := context.WithValue(context.Background(), "remote", "127.0.0.1:9999")
				e := CheckIpFrequent(cache, ctx)
				fmt.Println(i, e, "t0", len(cache.Keys()))
			}

			{
				ctx := context.WithValue(context.Background(), "remote", "127.0.0.2:2222")
				e := CheckIpFrequent(cache, ctx)
				fmt.Println(i, e, "t1", len(cache.Keys()))
			}

			{
				ctx := context.WithValue(context.Background(), "remote", "127.0.0.3:2222")
				e := CheckIpFrequent(cache, ctx)
				fmt.Println(i, e, "t2", len(cache.Keys()))
			}
		}
	}()

	time.Sleep(2 * time.Minute)

}
