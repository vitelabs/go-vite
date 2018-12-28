package discovery

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/log15"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
)

const maxNeighborsOneTrip = 6

var errIncompleteMsg = errors.New("send incomplete message")

type agent struct {
	self    *Node
	conn    *net.UDPConn
	peerKey ed25519.PrivateKey
	term    chan struct{}
	handler func(*packet)
	pool    pool
	wg      sync.WaitGroup
	log     log15.Logger
}

func (a *agent) start() {
	a.term = make(chan struct{})
	a.pool.start()

	a.wg.Add(1)
	common.Go(a.readLoop)
}

func (a *agent) stop() {
	if a.term == nil {
		return
	}

	if a.conn != nil {
		a.conn.Close()
	}

	select {
	case <-a.term:
	default:
		close(a.term)
		a.pool.stop()
		a.wg.Wait()
	}
}

func (a *agent) ping(id NodeID, addr *net.UDPAddr, ch chan<- bool) {
	hash, err := a.write(&Ping{
		ID:         a.self.ID,
		TCP:        a.self.TCP,
		Expiration: getExpiration(),
		Net:        a.self.Net,
		Ext:        a.self.Ext,
	}, addr)

	if err != nil {
		if ch != nil {
			ch <- false
		}

		return
	}

	a.pool.add(&wait{
		expectFrom: id,
		expectCode: pongCode,
		handler: &pingWait{
			sourceHash: hash,
			done:       ch,
		},
		expiration: getExpiration(),
	})
}

func (a *agent) pong(addr *net.UDPAddr, ack types.Hash) {
	msg := new(Pong)
	msg.ID = a.self.ID
	msg.Expiration = getExpiration()
	msg.Ping = ack
	a.write(msg, addr)
}

func (a *agent) findnode(id NodeID, addr *net.UDPAddr, target NodeID, need uint32, ch chan<- []*Node) {
	_, err := a.write(&FindNode{
		ID:         a.self.ID,
		Target:     target,
		Expiration: getExpiration(),
	}, addr)

	if err != nil {
		if ch != nil {
			ch <- nil
		}
		return
	}

	a.pool.add(&wait{
		expectFrom: id,
		expectCode: neighborsCode,
		handler: &findnodeWait{
			need: need,
			rec:  nil,
			ch:   ch,
		},
		expiration: getExpiration(),
	})
}

func (a *agent) sendNeighbors(addr *net.UDPAddr, nodes []*Node) {
	msg := new(Neighbors)
	msg.ID = a.self.ID
	msg.Expiration = getExpiration()

	// send nodes in batches
	carriage := make([]*Node, 0, maxNeighborsOneTrip)
	for _, node := range nodes {
		carriage = append(carriage, node)

		if len(carriage) == maxNeighborsOneTrip {
			msg.Nodes = carriage
			a.write(msg, addr)
			carriage = carriage[:0]
		}
	}

	if len(carriage) > 0 {
		msg.Nodes = carriage
		a.write(msg, addr)
	}
}

func (a *agent) readLoop() {
	defer a.wg.Done()
	defer a.conn.Close()

	buf := make([]byte, maxPacketLength)

	var tempDelay time.Duration
	var maxDelay = time.Second

	for {
		select {
		case <-a.term:
			return
		default:
		}

		n, addr, err := a.conn.ReadFromUDP(buf)

		if err != nil {
			if err, ok := err.(net.Error); ok && err.Temporary() {
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

			a.log.Error(fmt.Sprintf("udp read error %v", err))
			return
		}

		tempDelay = 0

		p, err := unPacket(buf[:n])
		if err != nil {
			a.log.Error(fmt.Sprintf("unpack message from %s error: %v", addr, err))
			continue
		}

		if p.msg.isExpired() {
			a.log.Warn(fmt.Sprintf("message %s from %s expired", p.msg, addr))
			continue
		}

		p.from = addr

		want := true
		if p.code == pongCode || p.code == neighborsCode {
			want = a.pool.rec(p)
		}

		if want {
			a.handler(p)
		}
	}
}

func (a *agent) write(msg Message, addr *net.UDPAddr) (hash types.Hash, err error) {
	data, hash, err := msg.pack(a.peerKey)

	if err != nil {
		a.log.Warn(fmt.Sprintf("pack message %s to %s error: %v", msg, addr, err))
		return
	}

	n, err := a.conn.WriteToUDP(data, addr)

	if err != nil {
		a.log.Warn(fmt.Sprintf("write message %s to %s error: %v", msg, addr, err))
		return
	}

	if n != len(data) {
		err = errIncompleteMsg
		a.log.Warn(fmt.Sprintf("write message %s to %s error: %v", msg, addr, err))
		return
	}

	return
}
