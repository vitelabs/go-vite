/*
 * Copyright 2019 The go-vite Authors
 * This file is part of the go-vite library.
 *
 * The go-vite library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The go-vite library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with the go-vite library. If not, see <http://www.gnu.org/licenses/>.
 */

package net

import (
	"errors"
	"fmt"
	"io"
	_net "net"
	"sync/atomic"
	"time"
)

type MockCodec struct {
	name     string
	r        chan Msg
	w        chan Msg
	rtimeout time.Duration
	wtimeout time.Duration
	term     chan struct{}
	closed   int32
	write    int32
}

type mockAddress struct {
	name string
}

func (m mockAddress) Network() string {
	return "mock codec"
}

func (m mockAddress) String() string {
	return "mock codec " + m.name
}

func MockPipe() (c1, c2 Codec) {
	chan1 := make(chan Msg)
	chan2 := make(chan Msg)

	return &MockCodec{
			name: "mock1",
			r:    chan1,
			w:    chan2,
			term: make(chan struct{}),
		}, &MockCodec{
			name: "mock2",
			r:    chan2,
			w:    chan1,
			term: make(chan struct{}),
		}
}

func (m *MockCodec) ReadMsg() (msg Msg, err error) {
	var ok bool

	var timer <-chan time.Time
	if m.rtimeout > 0 {
		timer = time.After(m.rtimeout)
	}

	select {
	case <-timer:
		return msg, errors.New("read timeout")

	case msg, ok = <-m.r:
		if ok {
			return msg, nil
		}

		return msg, io.EOF

	case <-m.term:
		err = fmt.Errorf("mock codec %s closed", m.name)
		return
	}
}

func (m *MockCodec) WriteMsg(msg Msg) (err error) {
	if atomic.LoadInt32(&m.closed) == 1 {
		return fmt.Errorf("mock codec %s closed", m.name)
	}

	atomic.AddInt32(&m.write, 1)
	defer atomic.AddInt32(&m.write, -1)

	var timer <-chan time.Time
	if m.wtimeout > 0 {
		timer = time.After(m.wtimeout)
	}

	select {
	case <-timer:
		return errors.New("write timeout")
	case m.w <- msg:
		return nil
	case <-m.term:
		err = fmt.Errorf("mock codec %s closed", m.name)
		return
	}
}

func (m *MockCodec) Close() error {
	if atomic.CompareAndSwapInt32(&m.closed, 0, 1) {
		close(m.term)

		for {
			if atomic.LoadInt32(&m.write) == 0 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}

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

func (m *MockCodec) Address() _net.Addr {
	return mockAddress{m.name}
}
