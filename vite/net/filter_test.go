package net

import (
	"testing"

	"time"

	"github.com/vitelabs/go-vite/common/types"
)

func TestHold(t *testing.T) {
	hash, e := types.HexToHash("8d9cef33f1c053f976844c489fc642855576ccd535cf2648412451d783147394")
	if e != nil {
		panic(e)
	}

	f := newFilter()

	// first time, should not hold
	if f.hold(hash) {
		t.Fail()
	}

	// have`nt done, then hold 2*timeThreshold and maxMark*2 times
	for i := 0; i < maxMark*2; i++ {
		if !f.hold(hash) {
			t.Fail()
		}
	}

	time.Sleep(timeThreshold)
	if !f.hold(hash) {
		t.Fail()
	}

	// have wait exceed 2 * timeThreshold and maxMark times from add
	time.Sleep(timeThreshold)

	if f.hold(hash) {
		t.Fail()
	}
}

func TestDone(t *testing.T) {
	hash, e := types.HexToHash("8d9cef33f1c053f976844c489fc642855576ccd535cf2648412451d783147394")
	if e != nil {
		panic(e)
	}

	f := newFilter()

	// first time, should hold
	if f.hold(hash) {
		t.Fail()
	}

	f.done(hash)

	// task done, then hold maxMark times and timeThreshold
	for i := 0; i < maxMark; i++ {
		if !f.hold(hash) {
			t.Fail()
		}
	}

	// have done, then hold timeThreshold
	time.Sleep(timeThreshold)
	if f.hold(hash) {
		t.Fail()
	}
}
