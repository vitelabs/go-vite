package p2p

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	crand "crypto/rand"
	"fmt"
	"io"
	"math/big"
	mrand "math/rand"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"

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

	//snappy.NewReader()
	//snappy.NewBufferedWriter()
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

func TestAccountBlock_Snappy(t *testing.T) {
	accountAddress1, privateKey, _ := types.CreateAddress()
	accountAddress2, _, _ := types.CreateAddress()

	block := &ledger.AccountBlock{
		PrevHash:       types.Hash{},
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: accountAddress1,
		ToAddress:      accountAddress2,
		Amount:         big.NewInt(1000),
		TokenId:        ledger.ViteTokenId,
		Height:         123,
		Quota:          1,
		Fee:            big.NewInt(0),
		PublicKey:      privateKey.PubByte(),
		Data:           []byte{'a', 'b', 'c', 'd', 'e'},
		LogHash:        &types.Hash{},
		Nonce:          []byte("test nonce test nonce"),
		Signature:      []byte("test signature test signature test signature"),
	}

	hash := block.ComputeHash()
	buf, err := block.Serialize()
	if err != nil {
		panic(err)
	}

	buf2 := snappy.Encode(nil, buf)
	fmt.Println(len(buf), len(buf2))

	buf3, err := snappy.Decode(nil, buf2)
	if err != nil {
		panic(err)
	}

	block2 := new(ledger.AccountBlock)
	err = block2.Deserialize(buf3)
	if err != nil {
		panic(err)
	}
	hash2 := block2.ComputeHash()

	if hash != hash2 {
		panic("different hash")
	}
}
