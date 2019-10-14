package vm

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/vm/util"
	"testing"
)

func TestMemoryGasCost(t *testing.T) {
	vm := &VM{gasTable: util.QuotaTableByHeight(1)}
	size := uint64(0xffffffffe0)
	v, _, err := memoryGasCost(vm, &memory{}, size)
	if err != nil {
		t.Error("didn't expect error:", err)
	}
	if v != 36028899963961341 {
		t.Errorf("Expected: 36028899963961341, got %d", v)
	}

	_, _, err = memoryGasCost(vm, &memory{}, size+1)
	if err == nil {
		t.Error("expected error")
	}

	_, _, err = memoryGasCost(vm, &memory{}, helper.MaxUint64-64)
	if err == nil {
		t.Errorf("Expected error")
	}
}
