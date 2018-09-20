package crypto

import (
	"testing"
	"encoding/hex"
)

func TestHash(t *testing.T) {
	data := "12343"
	hash0 := Hash(20, []byte(data))
	hash1 := Hash(32, []byte(data))
	println(hex.EncodeToString(hash0))
	println(hex.EncodeToString(hash1))
}
