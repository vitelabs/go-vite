package discovery

import (
	"fmt"
	"net"
	"sync"

	"github.com/vitelabs/go-vite/log15"
)

const maxMessageLength = 1200

type Socket interface {
	Start() error
	AddFilter(flt filter)
	WriteToUDP(b []byte, addr *net.UDPAddr) (int, error)
	Stop() error
}

type packet struct {
	payload []byte
	from    *net.UDPAddr
}

type filter = func(data []byte, from *net.UDPAddr) bool

type socket struct {
	*net.UDPConn
	addr *net.UDPAddr

	wg sync.WaitGroup

	// return false will interrupt the handle flow
	filter []filter

	handler func(pkt packet)

	queue chan packet

	log log15.Logger
}

func (s *socket) AddFilter(flt filter) {
	s.filter = append(s.filter, flt)
}

func (s *socket) Start() (err error) {
	s.UDPConn, err = net.ListenUDP("udp", s.addr)

	if err != nil {
		return err
	}

	s.queue = make(chan packet, 10)

	s.wg.Add(1)
	go s.readLoop()

	s.wg.Add(1)
	go s.handleLoop()

	return nil
}

func (s *socket) Stop() (err error) {
	err = s.UDPConn.Close()
	s.wg.Wait()

	return
}

func (s *socket) readLoop() {
	var n int
	var from *net.UDPAddr
	var err error

	defer s.wg.Done()

	defer close(s.queue)

Loop:
	for {
		var data = make([]byte, maxMessageLength)

		n, from, err = s.UDPConn.ReadFromUDP(data)

		if err != nil {
			s.log.Error(fmt.Sprintf("Failed to read from UDP socket: %v", err))
			return
		}

		for _, flt := range s.filter {
			if !flt(data[:n], from) {
				continue Loop
			}
		}

		s.queue <- packet{data[:n], from}
	}
}

func (s *socket) handleLoop() {
	defer s.wg.Done()

	for pkt := range s.queue {
		s.handler(pkt)
	}
}

func newSocket(listenAddress string, handler func(pkt packet), log log15.Logger) (Socket, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", listenAddress)
	if err != nil {
		return nil, err
	}

	var s = socket{
		addr:    udpAddr,
		handler: handler,
		log:     log,
	}

	return &s, nil
}
