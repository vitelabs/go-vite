package discovery

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/monitor"
	"net"
	"sync"
	"time"
)

var msgExpiration = 2 * time.Minute

// @section agent
var errStopped = errors.New("server has stopped")
var errTimeout = errors.New("waiting timeout")
var errUnmatchedPong = errors.New("unmatched pong")
var errUnsolicitedMsg = errors.New("unsolicited message")
var errMsgExpired = errors.New("message has expired")
var errUnkownMsg = errors.New("unknown message")

type udpAgent struct {
	maxNeighborsOneTrip int
	self                *Node
	conn                *net.UDPConn
	priv                ed25519.PrivateKey
	waiting             chan *wait
	res                 chan *res
	stopped             chan struct{}
	lock                sync.RWMutex
	running             bool
	packetHandler       func(*packet) error
}

// after send query. wating for reply.
type waitCallback func(Message) bool
type wait struct {
	expectFrom NodeID
	expectCode packetCode
	handleRes  waitCallback
	expiration time.Time
	errch      chan error
}
type wtList []*wait

func (wtl wtList) handle(rs *res) (rest wtList) {
	var err error
	for i, wt := range wtl {
		if wt.expectFrom == rs.from && wt.expectCode == rs.code {
			done := true
			if wt.handleRes != nil {
				done = wt.handleRes(rs.data)
			}

			rs.matched <- done
			wt.errch <- err

			if done {
				rest = append(wtl[:i], wtl[i+1:]...)
			}
			return
		}
	}

	return wtl
}

func (wtl wtList) clean() wtList {
	now := time.Now()
	i := 0
	for j := 0; j < len(wtl); j++ {
		wt := wtl[j]
		if wt.expiration.Before(now) {
			wt.errch <- errTimeout
		} else {
			wtl[i] = wt
			i++
		}
	}

	return wtl[:i]
}

type res struct {
	from    NodeID
	code    packetCode
	data    Message
	matched chan bool
}

func (d *udpAgent) start() {
	discvLog.Info("discv agent start")

	d.lock.Lock()
	defer d.lock.Unlock()

	if d.running {
		return
	}
	d.running = true

	go d.loop()
	go d.readLoop()
}

// implements table.agent interface
func (d *udpAgent) ping(node *Node) error {
	if node.ID == d.self.ID {
		return nil
	}

	hash, err := d.send(node.UDPAddr(), pingCode, &Ping{
		ID:         d.self.ID,
		IP:         d.self.IP,
		UDP:        d.self.UDP,
		TCP:        d.self.TCP,
		Expiration: getExpiration(),
	})

	if err != nil {
		return err
	}

	send := time.Now()

	return d.wait(node.ID, pongCode, func(m Message) bool {
		pong, ok := m.(*Pong)
		if ok && pong.Ping == hash {
			pongReceived := time.Now()
			monitor.LogDuration("p2p/discv", "ping", pongReceived.Sub(send).Nanoseconds())
			return true
		}
		return false
	})
}

func (d *udpAgent) pong(node *Node, ack types.Hash) error {
	if node.ID == d.self.ID {
		return nil
	}

	_, err := d.send(node.UDPAddr(), pongCode, &Pong{
		ID:         d.self.ID,
		Ping:       ack,
		Expiration: getExpiration(),
	})

	return err
}

func (d *udpAgent) findnode(n *Node, ID NodeID) (nodes []*Node, err error) {
	discvLog.Info(fmt.Sprintf("find %s to %s\n", ID, n))

	_, err = d.send(n.UDPAddr(), findnodeCode, &FindNode{
		ID:         d.self.ID,
		Target:     ID,
		Expiration: getExpiration(),
	})

	if err != nil {
		return nodes, err
	}
	findSend := time.Now()

	nodes = make([]*Node, 0, K)
	total := 0
	err = d.wait(n.ID, neighborsCode, func(m Message) bool {
		neighbors, ok := m.(*Neighbors)
		if ok {
			for _, n := range neighbors.Nodes {
				if n.Validate() == nil {
					nodes = append(nodes, n)
				}
				total++
			}
			monitor.LogDuration("p2p/discv", "findnode", time.Now().Sub(findSend).Nanoseconds())
			return total >= K
		}
		return false
	})

	return nodes, err
}

func (d *udpAgent) sendNeighbors(n *Node, nodes []*Node) error {
	neighbors := &Neighbors{
		ID:         d.self.ID,
		Expiration: getExpiration(),
	}

	carriage := make([]*Node, 0, d.maxNeighborsOneTrip)
	sent := false
	for _, node := range nodes {
		carriage = append(carriage, node)
		if len(carriage) == d.maxNeighborsOneTrip {
			neighbors.Nodes = carriage
			_, err := d.send(n.UDPAddr(), neighborsCode, neighbors)
			if err != nil {
				sent = true
			}
			carriage = carriage[:0]
		}
	}

	if !sent || len(carriage) > 0 {
		neighbors.Nodes = carriage
		d.send(n.UDPAddr(), neighborsCode, neighbors)
	}

	return nil
}

func (d *udpAgent) wait(ID NodeID, code packetCode, handleRes waitCallback) error {
	errch := make(chan error, 1)

	select {
	case d.waiting <- &wait{
		expectFrom: ID,
		expectCode: code,
		handleRes:  handleRes,
		expiration: time.Now().Add(watingTimeout),
		errch:      errch,
	}:
	case <-d.stopped:
		errch <- errStopped
	}

	return <-errch
}

func (d *udpAgent) stop() {
	discvLog.Info("discv agent stop")

	select {
	case <-d.stopped:
	default:
		close(d.stopped)
		d.conn.Close()
	}
}

func (d *udpAgent) loop() {
	var wtl wtList

	checkTicker := time.NewTicker(watingTimeout / 2)
	defer checkTicker.Stop()

loop:
	for {
		select {
		case <-d.stopped:
			for _, w := range wtl {
				w.errch <- errStopped
			}
			break loop
		case req := <-d.waiting:
			wtl = append(wtl, req)
		case res := <-d.res:
			wtl = wtl.handle(res)
			discvLog.Info(fmt.Sprintf("handle msg %s from %s\n", res.code, res.from))
		case <-checkTicker.C:
			wtl = wtl.clean()
		}
	}
}

func (d *udpAgent) readLoop() {
	buf := make([]byte, maxPacketLength)

loop:
	for {
		select {
		case <-d.stopped:
			break loop
		default:
		}

		n, addr, err := d.conn.ReadFromUDP(buf)

		p, err := unPacket(buf[:n])
		if err != nil {
			discvLog.Error(fmt.Sprintf("unpack msg from %s error: %v\n", addr, err))
			continue
		}
		p.from = addr

		err = d.packetHandler(p)
	}
}

func (d *udpAgent) send(addr *net.UDPAddr, code packetCode, m Message) (hash types.Hash, err error) {
	var data []byte
	data, hash, err = m.pack(d.priv)

	if err != nil {
		discvLog.Error(fmt.Sprintf("pack msg %s to %s error: %v\n", code, addr, err))
		return
	}

	_, err = d.conn.WriteToUDP(data, addr)
	if err != nil {
		discvLog.Error(fmt.Sprintf("send msg %s to %s error: %v\n", code, addr, err))
		return
	}

	discvLog.Info(fmt.Sprintf("send msg %s to %s done\n", code, addr))

	return
}

//func (d *udpAgent) receive(id NodeID, code packetCode, m Message) bool {
func (d *udpAgent) want(p *packet) bool {
	matched := make(chan bool, 1)
	select {
	case <-d.stopped:
		return false
	case d.res <- &res{
		from:    p.fromID,
		code:    p.code,
		data:    p.msg,
		matched: matched,
	}:
		return <-matched
	}
}
