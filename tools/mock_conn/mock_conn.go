package mock_conn

import (
	"errors"
	"io"
	"net"
	"sync/atomic"
	"time"
)

type mockConnAddress struct {
	name string
}

func (m mockConnAddress) Network() string {
	return "mock"
}

func (m mockConnAddress) String() string {
	return m.name
}

var zero = time.Time{}

type mockConn struct {
	closed   int32
	name     string
	rname    string
	read     <-chan byte
	write    chan<- byte
	term     chan struct{}
	rtimeout time.Time
	wtimeout time.Time
}

func (mc *mockConn) Read(b []byte) (n int, err error) {
	var timer <-chan time.Time
	if mc.rtimeout != zero {
		if time.Now().After(mc.rtimeout) {
			return 0, errors.New("read timeout")
		}

		timer = time.After(mc.rtimeout.Sub(time.Now()))
	}

	var ok bool
	for {
		select {
		case <-timer:
			return n, errors.New("read timeout")
		case b[n], ok = <-mc.read:
			if ok {
				n++
			} else {
				return n, io.EOF
			}

			if n == len(b) {
				return n, nil
			}
		case <-mc.term:
			return n, errors.New("mock conn closed")
		}
	}
}

func (mc *mockConn) Write(b []byte) (n int, err error) {
	var timer <-chan time.Time
	if mc.wtimeout != zero {
		if time.Now().After(mc.wtimeout) {
			return 0, errors.New("write timeout")
		}

		timer = time.After(mc.wtimeout.Sub(time.Now()))
	}

	for {
		select {
		case <-timer:
			return n, errors.New("write timeout")
		case mc.write <- b[n]:
			n++
			if n == len(b) {
				return n, nil
			}
		case <-mc.term:
			return n, errors.New("mock conn closed")
		}
	}
}

func (mc *mockConn) Close() error {
	if atomic.CompareAndSwapInt32(&mc.closed, 0, 1) {
		close(mc.term)
		return nil
	}

	return errors.New("mock conn closed")
}

func (mc *mockConn) LocalAddr() net.Addr {
	return mockConnAddress{
		name: mc.name,
	}
}

func (mc *mockConn) RemoteAddr() net.Addr {
	return mockConnAddress{
		name: mc.rname,
	}
}

func (mc *mockConn) SetDeadline(t time.Time) error {
	mc.rtimeout = t
	mc.wtimeout = t

	return nil
}

func (mc *mockConn) SetReadDeadline(t time.Time) error {
	mc.rtimeout = t
	return nil
}

func (mc *mockConn) SetWriteDeadline(t time.Time) error {
	mc.wtimeout = t
	return nil
}

func Pipe() (c1, c2 net.Conn) {
	ch1 := make(chan byte, 10)
	ch2 := make(chan byte, 10)

	c1, c2 = &mockConn{
		closed:   0,
		name:     "c1",
		rname:    "c2",
		read:     ch1,
		write:    ch2,
		term:     make(chan struct{}),
		rtimeout: time.Time{},
		wtimeout: time.Time{},
	}, &mockConn{
		closed:   0,
		name:     "c2",
		rname:    "c1",
		read:     ch2,
		write:    ch1,
		term:     make(chan struct{}),
		rtimeout: time.Time{},
		wtimeout: time.Time{},
	}

	return
}
