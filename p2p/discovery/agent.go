package discovery

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p/list"
)

const maxNeighborsOneTrip = 10

var errStopped = errors.New("discovery server has stopped")
var errWaitOvertime = errors.New("wait for response timeout")

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
	list list.List
	add  chan *wait
	rec  chan *packet
	term chan struct{}
	wg   sync.WaitGroup
}

func newWtPool() *wtPool {
	return &wtPool{
		list: list.New(),
		add:  make(chan *wait),
		rec:  make(chan *packet),
	}
}

func (p *wtPool) start() {
	p.term = make(chan struct{})

	p.wg.Add(1)
	common.Go(p.loop)
}

func (p *wtPool) stop() {
	if p.term == nil {
		return
	}

	select {
	case <-p.term:
	default:
		close(p.term)
		p.wg.Wait()
	}
}

func (p *wtPool) loop() {
	defer p.wg.Done()

	checkTicker := time.NewTicker(expiration)
	defer checkTicker.Stop()

	for {
		select {
		case <-p.term:
			p.list.Traverse(func(value interface{}) bool {
				wt, ok := value.(*wait)
				if ok {
					wt.handle(nil, errStopped, wt)
				}
				return true
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
	p.list.Filter(func(value interface{}) bool {
		wt, ok := value.(*wait)
		if ok && wt.expectFrom == rs.fromID && wt.expectCode == rs.code {
			return wt.handle(rs.msg, nil, wt)
		}
		return false
	})
}

func (p *wtPool) clean() {
	now := time.Now()
	p.list.Filter(func(value interface{}) bool {
		wt, ok := value.(*wait)
		if ok && wt.expiration.Before(now) {
			wt.handle(nil, errWaitOvertime, wt)
			return true
		}
		return false
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
type agentConfig struct {
	Self    *Node
	Addr    *net.UDPAddr
	Priv    ed25519.PrivateKey
	Handler func(*packet)
}

type agent struct {
	self       *Node
	addr       *net.UDPAddr
	conn       *net.UDPConn
	priv       ed25519.PrivateKey
	term       chan struct{}
	pktHandler func(*packet)
	pool       *wtPool
	wg         sync.WaitGroup
	write      chan *sendPkt
	read       chan *packet
	log        log15.Logger
}

func newAgent(cfg *agentConfig) *agent {
	return &agent{
		addr:       cfg.Addr,
		self:       cfg.Self,
		priv:       cfg.Priv,
		pktHandler: cfg.Handler,
		pool:       newWtPool(),
		write:      make(chan *sendPkt, 10),
		read:       make(chan *packet, 10),
		log:        log15.New("module", "p2p/agent"),
	}
}

// should run as goroutine
func (a *agent) start() error {
	a.log.Info("discovery agent start")

	udpConn, err := net.ListenUDP("udp", a.addr)
	if err != nil {
		return err
	}
	a.log.Info(fmt.Sprintf("udp listen at %s", a.addr))
	a.conn = udpConn

	a.term = make(chan struct{})
	a.pool.start()

	a.wg.Add(1)
	common.Go(a.readLoop)

	a.wg.Add(1)
	common.Go(a.writeLoop)

	a.wg.Add(1)
	common.Go(a.handleLoop)

	return nil
}

func (a *agent) stop() {
	if a.term == nil {
		return
	}

	if a.conn != nil {
		a.conn.Close()
	}

	discvLog.Info("discovery agent term")

	select {
	case <-a.term:
	default:
		close(a.term)
		a.pool.stop()
		a.wg.Wait()
	}
}

// implements discvAgent interface
func (a *agent) ping(node *Node, callback func(*Node, error)) {
	if node.ID == a.self.ID {
		return
	}

	a.send(&sendPkt{
		addr: node.UDPAddr(),
		code: pingCode,
		msg: &Ping{
			ID:         a.self.ID,
			IP:         a.self.IP,
			UDP:        a.self.UDP,
			TCP:        a.self.TCP,
			Net:        a.self.Net,
			Ext:        a.self.Ext,
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

func (a *agent) pong(node *Node, ack types.Hash) {
	if node.ID == a.self.ID {
		return
	}

	a.send(&sendPkt{
		addr: node.UDPAddr(),
		code: pongCode,
		msg: &Pong{
			ID:         a.self.ID,
			Ping:       ack,
			IP:         node.IP,
			Expiration: getExpiration(),
		},
	})
}

// should ping-pong checked before
func (a *agent) findnode(n *Node, ID NodeID, callback func([]*Node, error)) {
	a.send(&sendPkt{
		addr: n.UDPAddr(),
		code: findnodeCode,
		msg: &FindNode{
			ID:         a.self.ID,
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

				callback(neighbors.Nodes, nil)
				return true
			},
		},
	})
}

func (a *agent) sendNeighbors(n *Node, nodes []*Node) {
	neighbors := &Neighbors{
		ID:         a.self.ID,
		Expiration: getExpiration(),
	}

	// send nodes in batches
	carriage := make([]*Node, 0, maxNeighborsOneTrip)
	for _, node := range nodes {
		carriage = append(carriage, node)

		if len(carriage) == maxNeighborsOneTrip {
			neighbors.Nodes = carriage
			a.send(&sendPkt{
				addr: n.UDPAddr(),
				code: neighborsCode,
				msg:  neighbors,
			})
			carriage = carriage[:0]
		}
	}

	if len(carriage) > 0 {
		neighbors.Nodes = carriage
		a.send(&sendPkt{
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
			discvLog.Warn("discovery packet read channel is block")
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
				if s.wait != nil {
					s.wait.handle(nil, err, nil)
				}
				discvLog.Error(fmt.Sprintf("pack message %s to %s error: %v", s.msg, s.addr, err))
				continue
			}

			n, err := a.conn.WriteToUDP(data, s.addr)

			if err != nil {
				if s.wait != nil {
					s.wait.handle(nil, err, nil)
				}

				discvLog.Error(fmt.Sprintf("send message %s to %s error: %v", s.msg, s.addr, err))
			} else if n != len(data) {
				err = fmt.Errorf("send incomplete message %s (%d/%dbytes) to %s", s.msg, n, len(data), s.addr)
				if s.wait != nil {
					s.wait.handle(nil, err, nil)
				}
				discvLog.Error(err.Error())
			} else {
				monitor.LogEvent("p2p/discv", "send "+s.code.String())

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
