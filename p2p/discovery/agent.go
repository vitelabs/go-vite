package discovery

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p/list"
	"net"
	"sync"
	"time"
)

const maxNeighborsOneTrip = 10

var errStopped = errors.New("discovery server has stopped")
var errWaitOvertime = errors.New("wait for response timeout")
var errSendToSelf = errors.New("send discovery message to self")

// after send query. wating for reply.
type waitIsDone func(Message, error, *wait) bool

type wait struct {
	expectFrom NodeID
	expectCode packetCode
	sourceHash types.Hash
	handle     waitIsDone
	expiration time.Time
}

type wtPool struct {
	list *list.List
	add  chan *wait
	rec  chan *packet
}

func newWtPool() *wtPool {
	return &wtPool{
		list: list.New(),
		add:  make(chan *wait),
		rec:  make(chan *packet),
	}
}

func (p *wtPool) loop(stop <-chan struct{}) {
	checkTicker := time.NewTicker(expiration)
	defer checkTicker.Stop()

	for {
		select {
		case <-stop:
			p.list.Traverse(func(_, e *list.Element) {
				wt, _ := e.Value.(*wait)
				wt.handle(nil, errStopped, wt)
			})
			return
		case w := <-p.add:
			p.list.Append(w)
		case r := <-p.rec:
			p.handle(r)
		case <-checkTicker.C:
			p.clean()
		}
	}
}

func (p *wtPool) handle(rs *packet) {
	p.list.Traverse(func(prev, current *list.Element) {
		wt, _ := current.Value.(*wait)
		if wt.expectFrom == rs.fromID && wt.expectCode == rs.code {
			if wt.handle(rs.msg, nil, wt) {
				// remove current wait from list
				p.list.Remove(prev, current)
			}
		}
	})
}

func (p *wtPool) clean() {
	now := time.Now()
	p.list.Traverse(func(prev, current *list.Element) {
		wt, _ := current.Value.(*wait)
		if wt.expiration.Before(now) {
			wt.handle(nil, errWaitOvertime, wt)
			// remove current wait from list
			p.list.Remove(prev, current)
		}
	})
}

// @section send
type sendPkt struct {
	addr *net.UDPAddr
	code packetCode
	msg  Message
	wait *wait
}

// @section agent
type agent struct {
	self       *Node
	conn       *net.UDPConn
	priv       ed25519.PrivateKey
	term       chan struct{}
	pktHandler func(*packet)
	pool       *wtPool
	wg         sync.WaitGroup
	write      chan *sendPkt
	read       chan *packet
}

// should run as goroutine
func (d *agent) start() {
	discvLog.Info("discovery agent start")

	d.wg.Add(1)
	go func() {
		d.pool.loop(d.term)
		d.wg.Done()
	}()

	// should not run as goroutine
	d.wg.Add(1)
	go d.readLoop()

	d.wg.Add(1)
	go d.writeLoop()

	d.wg.Add(1)
	go d.handleLoop()
}

func (d *agent) stop() {
	discvLog.Info("discovery agent stop")

	close(d.term)
	d.wg.Wait()
}

// implements discvAgent interface
func (d *agent) ping(node *Node, callback func(*Node, error)) {
	if node.ID == d.self.ID {
		return
	}

	d.send(&sendPkt{
		addr: node.UDPAddr(),
		code: pingCode,
		msg: &Ping{
			ID:         d.self.ID,
			IP:         d.self.IP,
			UDP:        d.self.UDP,
			TCP:        d.self.TCP,
			Expiration: getExpiration(),
		},
		wait: &wait{
			expectFrom: node.ID,
			expectCode: pongCode,
			handle: func(m Message, err error, wt *wait) bool {
				defer callback(node, err)

				if err != nil {
					return false
				}

				pong, _ := m.(*Pong)
				if pong.Ping == wt.sourceHash {
					return true
				}

				err = errUnsolicitedMsg
				return false
			},
		},
	})
}

func (d *agent) pong(node *Node, ack types.Hash) {
	if node.ID == d.self.ID {
		return
	}

	d.send(&sendPkt{
		addr: node.UDPAddr(),
		code: pongCode,
		msg: &Pong{
			ID:         d.self.ID,
			Ping:       ack,
			IP:         node.IP,
			Expiration: getExpiration(),
		},
	})
}

// should ping-pong checked before
func (d *agent) findnode(n *Node, ID NodeID, callback func([]*Node, error)) {
	d.send(&sendPkt{
		addr: n.UDPAddr(),
		code: findnodeCode,
		msg: &FindNode{
			ID:         d.self.ID,
			Target:     ID,
			Expiration: getExpiration(),
		},
		wait: &wait{
			expectFrom: n.ID,
			expectCode: neighborsCode,
			handle: func(m Message, err error, _ *wait) bool {
				if err != nil {
					callback(nil, err)
					return true
				}

				neighbors, _ := m.(*Neighbors)

				total := 0
				nodes := make([]*Node, 0, maxNeighborsOneTrip)

				for _, n := range neighbors.Nodes {
					if n.Validate() == nil {
						nodes = append(nodes, n)
					}
					total++
				}

				callback(nodes, nil)
				return true
			},
		},
	})
}

func (d *agent) sendNeighbors(n *Node, nodes []*Node) {
	neighbors := &Neighbors{
		ID:         d.self.ID,
		Expiration: getExpiration(),
	}

	// send nodes in batches
	carriage := make([]*Node, 0, maxNeighborsOneTrip)
	sent := false
	for _, node := range nodes {
		carriage = append(carriage, node)

		if len(carriage) == maxNeighborsOneTrip {
			neighbors.Nodes = carriage
			d.send(&sendPkt{
				addr: n.UDPAddr(),
				code: neighborsCode,
				msg:  neighbors,
			})
			carriage = carriage[:0]
		}
	}

	// send nodes even if the list is empty
	if !sent || len(carriage) > 0 {
		neighbors.Nodes = carriage
		d.send(&sendPkt{
			addr: n.UDPAddr(),
			code: neighborsCode,
			msg:  neighbors,
		})
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
			discvLog.Error(fmt.Sprintf("udp read error %v", err))

			if err, ok := err.(net.Error); ok && err.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}

				if tempDelay > maxDelay {
					tempDelay = maxDelay
				}

				discvLog.Info(fmt.Sprintf("udp read tempError, wait %s", tempDelay))

				time.Sleep(tempDelay)

				continue
			}

			return
		}

		tempDelay = 0

		p, err := unPacket(buf[:n])
		if err != nil {
			discvLog.Error(fmt.Sprintf("unpack message from %s error: %v", addr, err))
			a.send(&sendPkt{
				addr: addr,
				code: exceptionCode,
				msg:  &Exception{Code: eCannotUnpack},
			})
			continue
		}

		p.from = addr

		if p.code == pongCode || p.code == neighborsCode {
			a.pool.rec <- p
		}

		select {
		case a.read <- p:
		default:
			discvLog.Error("discovery packet read channel is block")
		}
	}
}

func (a *agent) writeLoop() {
	defer a.wg.Done()

	for {
		select {
		case <-a.term:
			return
		case s := <-a.write:
			data, hash, err := s.msg.pack(a.priv)

			if err != nil {
				discvLog.Error(fmt.Sprintf("pack message %s to %s error: %v", s.msg, s.addr, err))
				continue
			}

			n, err := a.conn.WriteToUDP(data, s.addr)

			if err != nil {
				discvLog.Error(fmt.Sprintf("send message %s to %s error: %v", s.msg, s.addr, err))
			} else if n != len(data) {
				discvLog.Error(fmt.Sprintf("send incomplete message %s (%d/%dbytes) to %s", s.msg, n, len(data), s.addr))
			} else {
				monitor.LogEvent("p2p/discv", "send "+s.code.String())
				discvLog.Info(fmt.Sprintf("send message %s to %s done", s.msg, s.addr))

				if s.wait != nil {
					s.wait.sourceHash = hash
					s.wait.expiration = getExpiration()
					a.pool.add <- s.wait
				}
			}
		}
	}
}

func (a *agent) handleLoop() {
	defer a.wg.Done()

	for {
		select {
		case <-a.term:
			return
		case p := <-a.read:
			a.pktHandler(p)
		}
	}
}

func (a *agent) send(pkt *sendPkt) {
	a.write <- pkt
}
