package p2p

import (
	"bytes"
	crand "crypto/rand"
	"fmt"
	mrand "math/rand"
	"net"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/sha3"
)

func TestPutUint24(t *testing.T) {
	const maxLen = 8
	buf := make([]byte, maxLen)

	var i uint = 0

	m := PutVarint(buf, i)
	if m != 0 {
		t.Fatalf("0 should be 0 byte, but not %d bytes", m)
	}

	i = 1
	for offset := uint(0); offset < maxLen*8; offset++ {
		i = 1 << offset

		length := byte(offset/8 + 1)

		m = PutVarint(buf, i)

		if m > length {
			t.Fatalf("%d should be %d bytes, but not %d bytes", i, length, m)
		}
	}
}

func TestUint24(t *testing.T) {
	const maxLen = 8
	buf := make([]byte, maxLen)

	var i uint = 0
	m := PutVarint(buf, i)
	i2 := Varint(buf[:m])
	if i2 != i {
		t.Fatalf("put 0, but not get %d", i2)
	}

	i = 1
	for offset := uint(0); offset < maxLen*8; offset++ {
		i = 1 << offset

		m = PutVarint(buf, uint(i))

		i2 = Varint(buf[:m])
		if i2 != uint(i) {
			t.Fatalf("put %d, but get %d", i, i2)
		}
	}
}

func ExampleSHA3_Write() {
	hash := sha3.New256()

	buf := make([]byte, 10)
	_, err := crand.Read(buf)
	if err != nil {
		panic(err)
	}

	// write change the internal state, will get different sum result
	hash.Write(buf)
	hash1 := hash.Sum(nil)

	hash.Write(buf)
	hash2 := hash.Sum(nil)

	fmt.Println(bytes.Equal(hash1, hash2))
	// Output:
	// false
}

func ExampleSHA3_Reset() {
	hash := sha3.New256()
	hash1 := hash.Sum(nil)

	hash.Write([]byte("hello"))
	hash2 := hash.Sum(nil)

	hash.Reset() // Reset will clean the internal state
	hash3 := hash.Sum(nil)

	hash.Write([]byte("hello"))
	hash4 := hash.Sum(nil)

	fmt.Println(bytes.Equal(hash1, hash3), bytes.Equal(hash2, hash4))
	fmt.Println(bytes.Equal(hash1, hash2))
	// Output:
	// true true
	// false
}

func ExampleSHA3_Sum() {
	hash := sha3.New256()
	hash.Write([]byte("hello"))
	hash.Write([]byte("world"))
	hash1 := hash.Sum(nil)

	hash2 := hash.Sum([]byte("world2")) // append hash1 to `world2`
	hash3 := append([]byte("world2"), hash1...)

	fmt.Println(bytes.Equal(hash2, hash3))
	// Output:
	// true
}

func ExampleSHA3_Size() {
	hash := sha3.New256()
	hash.Write([]byte("hello"))
	size1 := hash.Size()
	hash1 := hash.Sum(nil)
	size2 := hash.Size()

	hash2 := hash.Sum([]byte("world"))
	size3 := hash.Size()

	fmt.Println(len(hash1) == size1)
	fmt.Println(size1 == size2, size1 == size3)
	fmt.Println(size1+5 == len(hash2))
	// Output:
	// true
	// true true
	// true
}

func MsgEqual(msg1, msg2 Msg) bool {
	if msg1.pid != msg2.pid {
		return false
	}
	if msg1.Code != msg2.Code {
		return false
	}
	if msg1.Id != msg2.Id {
		return false
	}

	return bytes.Equal(msg1.Payload, msg2.Payload)
}

func TestIdLength(t *testing.T) {
	var sizes = []byte{0, 1, 2, 4}
	var bits = []byte{0, 1, 2, 3}

	for i, size := range sizes {
		bit := idLengthToBits(size)
		if bit != bits[i] {
			t.Errorf("wrong length to bit %d --> %d", size, bit)
		}
		if bitsToIdLength(bit) != size {
			t.Errorf("wrong bit to length %d --> %d", bit, size)
		}
	}
}

func TestPutId(t *testing.T) {
	var ids = []MsgId{0, 222, 666, 77777}
	var lengths = []byte{0, 1, 2, 4}

	var buf = make([]byte, 4)
	for i, id := range ids {
		n := putId(id, buf)
		if n != lengths[i] {
			t.Errorf("wrong id %d length: %d", id, n)
		}
	}
}

func TestRetrieveMeta(t *testing.T) {
	var isizes = []byte{0, 1, 2, 4}
	var lsizes = []byte{0, 1, 2, 3}
	var cs = []bool{true, false}

	for _, isize := range isizes {
		for _, lsize := range lsizes {
			for _, c := range cs {
				meta := storeMeta(isize, lsize, c)
				isize2, lsize2, c2 := retrieveMeta(meta)
				if isize != isize2 {
					t.Errorf("wrong isize: %d %d", isize, isize2)
				}
				if lsize != lsize2 {
					t.Errorf("wrong lsize: %d %d", lsize, lsize2)
				}
				if c != c2 {
					t.Errorf("wrong compress: %v %v", c, c2)
				}
			}
		}
	}
}

func TestCodec(t *testing.T) {
	const addr = "127.0.0.1:10000"
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	channel := make(chan Msg, 1)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		conn, e := ln.Accept()
		if e != nil {
			panic(err)
		}

		c1 := NewTransport(conn, 100, readMsgTimeout, writeMsgTimeout)

		var msg Msg
		var werr error
		var buf []byte

		const total = 10000001
		const max = 1 << 8
		for i := 0; i < total; i *= 2 {
			buf = make([]byte, i)
			_, _ = crand.Read(buf)

			msg = Msg{
				pid:     ProtocolID(i % max),
				Code:    ProtocolID(i % max),
				Id:      uint32(i),
				Payload: buf,
			}

			werr = c1.WriteMsg(msg)
			fmt.Printf("write message %d/%d/%d, %d bytes\n", msg.pid, msg.Code, msg.Id, len(msg.Payload))

			if werr != nil {
				t.Fatalf("write error: %v", werr)
			}

			channel <- msg

			i++
		}

		fmt.Println("sent done")
		_ = conn.Close()
		_ = ln.Close()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		var msg, msg2 Msg
		var rerr error

		conn, rerr := net.Dial("tcp", addr)
		if rerr != nil {
			panic(rerr)
		}

		c2 := NewTransport(conn, 100, readMsgTimeout, writeMsgTimeout)

		for {
			msg, rerr = c2.ReadMsg()

			if rerr != nil {
				if strings.Index(rerr.Error(), "EOF") == -1 {
					t.Errorf("read error: %v", rerr)
				}
				break
			}

			msg2 = <-channel
			if MsgEqual(msg, msg2) {
				fmt.Printf("read message %d/%d/%d, %d bytes\n", msg.pid, msg.Code, msg.Id, len(msg.Payload))
			} else {
				t.Error("message not equal")
			}
		}

		_ = conn.Close()
	}()

	go func() {
		err = http.ListenAndServe("0.0.0.0:8080", nil)
		if err != nil {
			panic(err)
		}
	}()

	wg.Wait()
}

func BenchmarkMockCodec_WriteMsg(b *testing.B) {
	const addr = "127.0.0.1:10000"
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		conn, e := ln.Accept()
		if e != nil {
			panic(err)
		}

		c1 := NewTransport(conn, 100, readMsgTimeout, writeMsgTimeout)

		const payloadSize = 1000
		const max = 1 << 8

		var werr error
		for i := 0; i < b.N; i++ {
			size := mrand.Intn(payloadSize) + payloadSize
			buf := make([]byte, size)
			_, _ = crand.Read(buf)

			msg := Msg{
				pid:     ProtocolID(i % max),
				Code:    ProtocolID(i % max),
				Id:      uint32(i % maxPayloadSize),
				Payload: buf,
			}

			werr = c1.WriteMsg(msg)
			fmt.Printf("write %d message %d/%d, %d bytes\n", i, msg.pid, msg.Code, len(msg.Payload))

			if err != nil {
				b.Fatalf("write error: %v", werr)
			}
		}

		fmt.Println("sent done")
		_ = conn.Close()
		_ = ln.Close()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		var msg Msg
		var rerr error

		conn, rerr := net.Dial("tcp", addr)
		if rerr != nil {
			panic(rerr)
		}

		c2 := NewTransport(conn, 100, readMsgTimeout, writeMsgTimeout)

		for {
			msg, rerr = c2.ReadMsg()

			if rerr != nil {
				if strings.Index(rerr.Error(), "EOF") == -1 {
					b.Errorf("read error: %v", rerr)
				}
				break
			}

			fmt.Printf("read message %d/%d/%d, %d bytes\n", msg.pid, msg.Code, msg.Id, len(msg.Payload))
		}

		_ = conn.Close()
	}()

	wg.Wait()
}

/*
// will panic
func TestParallel(t *testing.T) {
	var m = make(map[string]int)

	t.Run("write", func(t *testing.T) {
		t.Parallel() // will run this test parallel
		for i := 0; i < 1000; i++ {
			m["hello"] = i
			time.Sleep(10 * time.Millisecond)
		}
	})

	t.Run("read", func(t *testing.T) {
		t.Parallel()
		for i := 0; i < 1000; i++ {
			t.Log(m["hello"])
			time.Sleep(10 * time.Millisecond)
		}
	})
}
*/
