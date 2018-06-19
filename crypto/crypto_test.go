package crypto

import (
	"testing"
	"encoding/hex"

)

const (
	LONG_BYTES = 100000
)

func TestHash256(t *testing.T) {

	println(hex.EncodeToString(Hash256([]byte{1, 2, 3})))
	println(hex.EncodeToString(Hash256([]byte{})))
	println(hex.EncodeToString(Hash256(nil)))
	LongBytes := make([]byte, LONG_BYTES)
	for i := 0; i < LONG_BYTES; i++ {
		LongBytes[i] = 21
	}
	println(hex.EncodeToString(Hash256(LongBytes)))

}


func TestGoSyntax(t *testing.T) {
	b := []byte{'1', '2', '3'}

	println(string(b[:2]))
	println(string(b[1:3]))
}
