package p2p

import (
	"crypto/rand"
	"testing"

	"github.com/golang/snappy"
)

func Test_Snappy_Encode(t *testing.T) {
	n := 10

	for {
		for i := 0; i < 10; i++ {
			buf := make([]byte, n)
			_, err := rand.Read(buf)
			if err != nil {
				panic(err)
			}
			buf2 := snappy.Encode(nil, buf)
			t.Log(n, len(buf2), len(buf2)/n)
		}

		n += 10
	}
}
