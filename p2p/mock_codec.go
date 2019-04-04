package p2p

import (
	"errors"
	"io"
	"sync/atomic"
	"time"
)

type MockCodec struct {
	r        chan Msg
	w        chan Msg
	rtimeout time.Duration
	wtimeout time.Duration
	closed   int32
}

func MockPipe() (c1, c2 Codec) {
	chan1 := make(chan Msg)
	chan2 := make(chan Msg)

	return &MockCodec{
			r: chan1,
			w: chan2,
		}, &MockCodec{
			r: chan2,
			w: chan1,
		}
}

func (m *MockCodec) ReadMsg() (msg Msg, err error) {
	var ok bool
	now := time.Now()
	if now.Add(m.rtimeout) == now {
		msg, ok = <-m.r
	} else {
		select {
		case <-time.After(m.rtimeout):
			return msg, errors.New("read timeout")
		case msg, ok = <-m.r:
		}
	}

	if ok {
		return msg, nil
	}

	return msg, io.EOF
}

func (m *MockCodec) WriteMsg(msg Msg) error {
	if atomic.LoadInt32(&m.closed) == 1 {
		return errors.New("closed")
	}

	now := time.Now()
	if now.Add(m.rtimeout) == now {
		m.w <- msg
	} else {
		select {
		case <-time.After(m.wtimeout):
			return errors.New("write timeout")
		case m.w <- msg:
		}
	}

	return nil
}

func (m *MockCodec) Close() error {
	if atomic.CompareAndSwapInt32(&m.closed, 0, 1) {
		time.Sleep(100 * time.Minute)
		close(m.r)
		close(m.w)

		return nil
	}

	return errors.New("closed")
}

func (m *MockCodec) SetReadTimeout(timeout time.Duration) {
	m.rtimeout = timeout
}

func (m *MockCodec) SetWriteTimeout(timeout time.Duration) {
	m.wtimeout = timeout
}

func (m *MockCodec) SetTimeout(timeout time.Duration) {
	m.rtimeout = timeout
	m.wtimeout = timeout
}

func (m *MockCodec) Address() string {
	return "mock"
}
