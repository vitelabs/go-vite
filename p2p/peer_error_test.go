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
