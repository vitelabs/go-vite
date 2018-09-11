package vm

import (
	"math/big"
	"testing"
)

func TestIntPoolPoolGet(t *testing.T) {
	poolOfIntPools.pools = make([]*intPool, 0, poolDefaultCap)

	nip := poolOfIntPools.get()
	if nip == nil {
		t.Fatalf("Invalid pool allocation")
	}
}

func TestIntPoolPoolPut(t *testing.T) {
	poolOfIntPools.pools = make([]*intPool, 0, poolDefaultCap)

	nip := poolOfIntPools.get()
	if len(poolOfIntPools.pools) != 0 {
		t.Fatalf("Pool got added to list when none should have been")
	}

	poolOfIntPools.put(nip)
	if len(poolOfIntPools.pools) == 0 {
		t.Fatalf("Pool did not get added to list when one should have been")
	}
}

func TestIntPoolPoolReUse(t *testing.T) {
	poolOfIntPools.pools = make([]*intPool, 0, poolDefaultCap)
	nip := poolOfIntPools.get()
	poolOfIntPools.put(nip)
	poolOfIntPools.get()

	if len(poolOfIntPools.pools) != 0 {
		t.Fatalf("Invalid number of pools. Got %d, expected %d", len(poolOfIntPools.pools), 0)
	}
}

func TestIntPool(t *testing.T) {
	pool := newIntPool()
	if pool.get() == nil {
		t.Fatalf("Get element from empty pool failed")
	}
	pool.put(big.NewInt(1))
	if pool.get() == nil {
		t.Fatalf("Get element from non-empty pool failed")
	}

	if pool.getZero().Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("Get zero from pool failed")
	}
}
