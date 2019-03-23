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

package p2p

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/tools/ticket"

	"github.com/vitelabs/go-vite/log15"
)

var (
	errServerNotStarted     = errors.New("server is not start")
	errServerAlreadyStarted = errors.New("server has already started")
)

const retryStartDuration = 5 * time.Second
const retryStartCount = 5

type Server interface {
	Start() error
	Stop() error
}

type server struct {
	running int32 // atomic

	retryStartDuration time.Duration // restart duration
	retryStartCount    int           // max restart count

	wg sync.WaitGroup // Wait for all jobs done

	maxInbound int
	maxPending int

	hkr Handshaker
	pm  peerManager

	listenAddress string
	ln            net.Listener
	tkt           ticket.Ticket
	log           log15.Logger
}

func newServer(rd time.Duration, rc, maxi, maxp int, hkr Handshaker, pm peerManager, addr string, log log15.Logger) Server {
	return &server{
		retryStartDuration: rd,
		retryStartCount:    rc,
		maxInbound:         maxi,
		maxPending:         maxp,
		hkr:                hkr,
		pm:                 pm,
		listenAddress:      addr,
		tkt:                ticket.New(maxp),
		log:                log,
	}
}

func (srv *server) Start() error {
	if atomic.CompareAndSwapInt32(&srv.running, 0, 1) {
		ln, err := net.Listen("tcp", srv.listenAddress)
		if err != nil {
			return err
		}

		srv.ln = ln

		srv.wg.Add(1)
		go srv.loop()

		return nil
	}

	return errServerAlreadyStarted
}

func (srv *server) Stop() error {
	if atomic.CompareAndSwapInt32(&srv.running, 1, 0) {
		err := srv.ln.Close()
		srv.wg.Wait()
		return err
	}
	return errServerNotStarted
}

func (srv *server) loop() {
	defer srv.wg.Done()

	srv.tkt.Reset()

	var tempDelay time.Duration
	var maxDelay = time.Second

	var err error

	for {
		// will return after handshake
		srv.tkt.Take()

		var conn net.Conn
		conn, err = srv.ln.Accept()

		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}

				if tempDelay > maxDelay {
					tempDelay = maxDelay
				}

				time.Sleep(tempDelay)

				continue
			}
			break
		}

		srv.wg.Add(1)
		go srv.handle(conn)
	}

	_ = srv.tkt.Close()

	srv.onStopListen(err)
}

func (srv *server) handle(c net.Conn) {
	defer srv.wg.Done()

	p, err := srv.hkr.Handshake(c, Inbound)

	srv.tkt.Return()

	if err != nil {
		srv.pm.register(p)
	} else {
		srv.log.Error(fmt.Sprintf("handshake with peer %s error: %v", c.RemoteAddr(), err))
	}
}

func (srv *server) onStopListen(err error) {
	srv.log.Warn(fmt.Sprintf("stop listen: %v", err))

	time.AfterFunc(srv.retryStartDuration, srv.retryListen)
}

func (srv *server) retryListen() {
	if atomic.LoadInt32(&srv.running) == 1 {
		srv.loop()
	}
}
