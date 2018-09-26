package ledger

import (
	"bytes"
	"testing"
	"time"
)

//func TestGetGenesesSnapshotBlock(t *testing.T) {
//
//}

func BenchmarkGetGenesisSnapshotBlock(b *testing.B) {
	aBytes := []byte{123, 23, 224}
	for i := 0; i < 100000000; i++ {
		var aTime = time.Unix(12123123123133123, 0)
		noThing(bytes.Equal(aBytes, []byte(string(aTime.Unix()))))
	}

}

func noThing(interface{}) {

}
