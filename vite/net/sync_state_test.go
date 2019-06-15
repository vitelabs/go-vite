package net

import (
	"fmt"
	"testing"
)

func ExampleSyncState_MarshalText() {
	var s SyncState

	data, err := s.MarshalText()
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", data)
	// Output:
	// Sync Not Start
}

func TestSyncState_MarshalText(t *testing.T) {
	var s SyncState

	data, err := s.MarshalText()
	if err != nil {
		panic(err)
	}

	var s2 = new(SyncState)
	err = s2.UnmarshalText(data)
	if err != nil {
		panic(err)
	}

	if *s2 != s {
		t.Error("wrong state", *s2)
	}
}
