package crypto

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestHash(t *testing.T) {
	data := "12343"
	hash0 := Hash(20, []byte(data))
	hash1 := Hash(32, []byte(data))
	println(hex.EncodeToString(hash0))
	println(hex.EncodeToString(hash1))

	a := []byte{1, 2}
	b := []byte{3, 4}
	fmt.Println(hex.EncodeToString(Hash256(a, b)))
	fmt.Println(hex.EncodeToString(Hash256([]byte{1, 2, 3, 4})))

	if !bytes.Equal(Hash256(a, b), Hash256([]byte{1, 2, 3, 4})) {
		t.Fatal("not equal")
	}
}

func BenchmarkHash256(b *testing.B) {
	buf := make([]byte, 4096)
	_, _ = rand.Read(buf)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Hash256(buf)
	}
}
