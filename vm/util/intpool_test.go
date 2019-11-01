package util

import (
	"math/big"
	"testing"
)

func TestIntPoolPoolGet(t *testing.T) {
	PoolOfIntPools.pools = make([]*IntPool, 0, poolDefaultCap)

	nip := PoolOfIntPools.Get()
	if nip == nil {
		t.Fatalf("Invalid pool allocation")
	}
}

func TestIntPoolPoolPut(t *testing.T) {
	PoolOfIntPools.pools = make([]*IntPool, 0, poolDefaultCap)

	nip := PoolOfIntPools.Get()
	if len(PoolOfIntPools.pools) != 0 {
		t.Fatalf("Pool got added to list when none should have been")
	}

	PoolOfIntPools.Put(nip)
	if len(PoolOfIntPools.pools) == 0 {
		t.Fatalf("Pool did not get added to list when one should have been")
	}
}

func TestIntPoolPoolReUse(t *testing.T) {
	PoolOfIntPools.pools = make([]*IntPool, 0, poolDefaultCap)
	nip := PoolOfIntPools.Get()
	PoolOfIntPools.Put(nip)
	PoolOfIntPools.Get()

	if len(PoolOfIntPools.pools) != 0 {
		t.Fatalf("Invalid number of pools. Got %d, expected %d", len(PoolOfIntPools.pools), 0)
	}
}

func TestIntPool(t *testing.T) {
	pool := newIntPool()
	if pool.Get() == nil {
		t.Fatalf("Get element from empty pool failed")
	}
	pool.Put(big.NewInt(1))
	if pool.Get() == nil {
		t.Fatalf("Get element from non-empty pool failed")
	}

	if pool.GetZero().Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("Get zero from pool failed")
	}
}
