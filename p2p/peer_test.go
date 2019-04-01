package p2p

import (
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

type mockConn struct {
	w chan []byte
	r chan []byte
}

func (mc *mockConn) Read(b []byte) (n int, err error) {
	panic(nil)
}

func (mc *mockConn) Write(b []byte) (n int, err error) {
	panic("implement me")
}

func (mc *mockConn) Close() error {
	panic("implement me")
}

func (mc *mockConn) LocalAddr() net.Addr {
	panic("implement me")
}

func (mc *mockConn) RemoteAddr() net.Addr {
	panic("implement me")
}

func (mc *mockConn) SetDeadline(t time.Time) error {
	panic("implement me")
}

func (mc *mockConn) SetReadDeadline(t time.Time) error {
	panic("implement me")
}

func (mc *mockConn) SetWriteDeadline(t time.Time) error {
	panic("implement me")
}

func Benchmark_Atomic(b *testing.B) {
	var a int32
	for i := 0; i < b.N; i++ {
		fmt.Println(atomic.LoadInt32(&a))
	}
}

func Benchmark_NoAtomic(b *testing.B) {
	var a int32
	for i := 0; i < b.N; i++ {
		fmt.Println(a)
	}
}
