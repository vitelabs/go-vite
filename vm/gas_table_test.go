package vm

import (
	"github.com/vitelabs/go-vite/vm/util"
	"testing"
)

func TestMemoryGasCost(t *testing.T) {
	size := uint64(0xffffffffe0)
	v, err := memoryGasCost(&memory{}, size)
	if err != nil {
		t.Error("didn't expect error:", err)
	}
	if v != 36028899963961341 {
		t.Errorf("Expected: 36028899963961341, got %d", v)
	}

	_, err = memoryGasCost(&memory{}, size+1)
	if err == nil {
		t.Error("expected error")
	}

	_, err = memoryGasCost(&memory{}, util.MaxUint64-64)
	if err == nil {
		t.Errorf("Expected error")
	}
}
