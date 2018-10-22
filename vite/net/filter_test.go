package net

import (
	"testing"

	"time"

	"github.com/vitelabs/go-vite/common/types"
)

func TestDone(t *testing.T) {
	hashes, e := types.HexToHash("8d9cef33f1c053f976844c489fc642855576ccd535cf2648412451d783147394")
	if e != nil {
		panic(e)
	}
	f := newFilter()
	println(f.hold(hashes))
	println(f.hold(hashes))
	println(f.has(hashes))
	f.done(hashes)
	for {
		println(f.hold(hashes), "hold")
		println(f.has(hashes), "has")
		time.Sleep(time.Second)
	}
}
