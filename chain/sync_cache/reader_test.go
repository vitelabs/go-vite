package sync_cache

import (
	"fmt"
	"testing"

	"github.com/golang/snappy"
)

func TestSnappy_Decode(t *testing.T) {
	buf := make([]byte, 50)
	buf[0] = 10

	buf2 := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	buf3 := snappy.Encode(buf[1:], buf2)
	for i, b := range buf3 {
		if buf[i+1] != b {
			t.Fail()
		}
	}

	fmt.Println(buf)

	buf4, err := snappy.Decode(buf[1:], buf[1:11])
	if err != nil {
		panic(err)
	}

	fmt.Println(buf4)
}
