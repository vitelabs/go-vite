package p2p

import (
	"bytes"
	"crypto/rand"
	"os"
	"sync"
	"testing"

	"github.com/golang/snappy"
)

func TestSnappy_Encode(t *testing.T) {
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
		if n > 10000 {
			break
		}
	}
}

func TestSnappy_Reader(t *testing.T) {
	c1, c2, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	reader := snappy.NewReader(c1)
	writer := snappy.NewBufferedWriter(c2)

	const total = 10000
	var buf = make([]byte, total)
	_, _ = rand.Read(buf)
	var buf2 = snappy.Encode(nil, buf)
	var buf3 = make([]byte, len(buf2))

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		start := 0
		const chunk = 1000

		for start < total {
			stop := start + chunk
			if stop > total {
				stop = total
			}

			n, err := writer.Write(buf[start:stop])
			if err != nil {
				panic(err)
			}
			if n != stop-start {
				panic("write too short")
			}

			start = stop
		}

		_ = writer.Close()
		//_ = c1.Close()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		start := 0
		const chunk = 2000

		for start < len(buf3) {
			stop := start + chunk
			if stop > len(buf3) {
				stop = len(buf3)
			}

			n, err := reader.Read(buf3[start:stop])
			if err != nil {
				start += n
				break
			}

			start += n
		}

		buf2 = buf3[:start]
	}()

	wg.Wait()

	if !bytes.Equal(buf2, buf3) {
		t.Errorf("diff compress: %d %d", len(buf2), len(buf3))
	}
}
