package p2p

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	crand "crypto/rand"
	"fmt"
	"io"
	mrand "math/rand"
	"testing"

	"github.com/golang/snappy"
)

func BenchmarkSnappy_Encode(b *testing.B) {
	var buf, dest []byte
	var l int
	var err error
	var s, d int
	for i := 0; i < b.N; i++ {
		l = mrand.Intn(10000)
		buf = make([]byte, l)
		_, err = io.ReadFull(crand.Reader, buf)
		if err != nil {
			b.Fatal(err)
		}
		s += l
		dest = snappy.Encode(nil, buf)
		d += len(dest)
	}
	fmt.Printf("snappy: %f\n", float64(d)/float64(s))
}

func BenchmarkSnappy_Decode(b *testing.B) {

}

func BenchmarkZlib_Encode(b *testing.B) {
	var buf []byte
	var dest = make([]byte, 100000)
	var l int
	var err error
	var s, d int
	var bf *bytes.Buffer

	for i := 0; i < b.N; i++ {
		l = mrand.Intn(10000)
		buf = make([]byte, l)
		_, err = io.ReadFull(crand.Reader, buf)
		if err != nil {
			b.Fatal(err)
		}
		s += l

		bf = bytes.NewBuffer(dest)
		w := zlib.NewWriter(bf)
		_, err = w.Write(buf)
		if err != nil {
			b.Fatal(err)
		}
		err = w.Close()
		if err != nil {
			b.Fatal(err)
		}

		d += len(bf.Bytes())
	}

	fmt.Printf("zlib: %f\n", float64(d)/float64(s))
}

func BenchmarkZlib_Decode(b *testing.B) {

}

func BenchmarkGzip_Encode(b *testing.B) {
	var buf []byte
	var dest = make([]byte, 100000)
	var l int
	var err error
	var s, d int
	var bf *bytes.Buffer

	for i := 0; i < b.N; i++ {
		l = mrand.Intn(10000)
		buf = make([]byte, l)
		_, err = io.ReadFull(crand.Reader, buf)
		if err != nil {
			b.Fatal(err)
		}
		s += l

		bf = bytes.NewBuffer(dest)
		w := gzip.NewWriter(bf)
		_, err = w.Write(buf)
		if err != nil {
			b.Fatal(err)
		}
		err = w.Close()
		if err != nil {
			b.Fatal(err)
		}

		d += len(bf.Bytes())
	}

	fmt.Printf("gzip: %f\n", float64(d)/float64(s))
}

func BenchmarkGzip_Decode(b *testing.B) {

}
