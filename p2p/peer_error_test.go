package p2p

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestPeerError_String(t *testing.T) {
	var n = rand.Int()
	var p = PeerError(n)
	fmt.Println(p.String(), p.Error())
}

func TestPeerError_UnknownReason(t *testing.T) {
	var e = PeerUnknownReason
	var b = byte(e)
	fmt.Printf("%v\n", e)
	if b != 255 {
		t.Fail()
	}
}
