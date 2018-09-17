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

var watingTimeout = 10 * time.Second // must be enough little, at least than checkInterval

// after send query. wating for reply.
type waitIsDone func(Message) bool

type wait struct {
	expectFrom NodeID
	expectCode packetCode
	handle     waitIsDone
	expiration time.Time
	errch      chan error
}

type res struct {
	from    NodeID
	code    packetCode
	data    Message
	matched chan bool
}

type wtList struct {
	list []*wait
	term chan struct{}
	add  chan *wait
	rec  chan *res
}

func newWtList() *wtList {
	wt := &wtList{
		term: make(chan struct{}),
		add:  make(chan *wait),
		rec:  make(chan *res),
	}

	go wt.loop()

	return wt
}

func (wtl *wtList) loop() {
	checkTicker := time.NewTicker(watingTimeout / 2)
	defer checkTicker.Stop()

loop:
	for {
		select {
		case <-wtl.term:
			for _, wt := range wtl.list {
				wt.errch <- errStopped
			}
			break loop
		case w := <-wtl.add:
			wtl.list = append(wtl.list, w)
		case r := <-wtl.rec:
			wtl.handle(r)
		case <-checkTicker.C:
			wtl.clean()
		}
	}
}

func (wtl *wtList) stop() {
	select {
	case <-wtl.term:
	default:
		close(wtl.term)
	}
}

func (wtl *wtList) handle(rs *res) {
	var err error
	for i, wt := range wtl.list {
		if wt.expectFrom == rs.from && wt.expectCode == rs.code {
			done := true
			if wt.handle != nil {
				done = wt.handle(rs.data)
			}

			rs.matched <- done
			wt.errch <- err

			if done {
				wtl.list = append(wtl.list[:i], wtl.list[i+1:]...)
				wtl.list = wtl.list[:len(wtl.list)-1]
			}
			return
		}
	}
}

func (wtl *wtList) clean() {
	now := time.Now()
	i := 0
	for _, wt := range wtl.list {
		if wt.expiration.Before(now) {
			wt.errch <- errWaitOvertime
		} else {
			wtl.list[i] = wt
			i++
		}
	}
}

// @section agent
var errStopped = errors.New("discovery server has stopped")
var errWaitOvertime = errors.New("wait for response timeout")

type udpAgent struct {
	maxNeighborsOneTrip int
	self                *Node
	conn                *net.UDPConn
	priv                ed25519.PrivateKey
	term                chan struct{}
	lock                sync.RWMutex
	running             bool
	packetHandler       func(*packet) error
	wtl                 *wtList
}

// should run as goroutine
func (d *udpAgent) start() {
	discvLog.Info("discv agent start")

	d.lock.Lock()
	defer d.lock.Unlock()

	if d.running {
		return
	}
	d.running = true

	d.wtl = newWtList()

	// should not run as goroutine
	d.readLoop()

	d.wtl.stop()
}

// implements discvAgent interface
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
			monitor.LogDuration("p2p/discv", "ping", time.Now().Sub(send).Nanoseconds())
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

// should ping-pong checked before
func (d *udpAgent) findnode(n *Node, ID NodeID) (nodes []*Node, err error) {
	discvLog.Info(fmt.Sprintf("find %s to %s\n", ID, n))

	_, err = d.send(n.UDPAddr(), findnodeCode, &FindNode{
		ID:         d.self.ID,
		Target:     ID,
		Expiration: getExpiration(),
	})

	if err != nil {
		return
	}

	send := time.Now()

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
			monitor.LogDuration("p2p/discv", "findnode", time.Now().Sub(send).Nanoseconds())
			return total >= K
		}
		return false
	})

	return
}

func (d *udpAgent) sendNeighbors(n *Node, nodes []*Node) (err error) {
	neighbors := &Neighbors{
		ID:         d.self.ID,
		Expiration: getExpiration(),
	}

	// send nodes in batches
	carriage := make([]*Node, 0, d.maxNeighborsOneTrip)
	sent := false
	for _, node := range nodes {
		carriage = append(carriage, node)

		if len(carriage) == d.maxNeighborsOneTrip {
			neighbors.Nodes = carriage
			_, err = d.send(n.UDPAddr(), neighborsCode, neighbors)

			if err != nil {
				sent = true
				discvLog.Info(fmt.Sprintf("send %d neighbors to %s\n", len(carriage), n))
			} else {
				discvLog.Error(fmt.Sprintf("send %d neighbors to %s error: %v\n", len(carriage), n, err))
			}
			carriage = carriage[:0]
		}
	}

	// send nodes even if the list is empty
	if !sent || len(carriage) > 0 {
		neighbors.Nodes = carriage
		_, err = d.send(n.UDPAddr(), neighborsCode, neighbors)

		if err != nil {
			discvLog.Info(fmt.Sprintf("send %d neighbors to %s\n", len(carriage), n))
		} else {
			discvLog.Error(fmt.Sprintf("send %d neighbors to %s error: %v\n", len(carriage), n, err))
		}
	}

	return
}

func (d *udpAgent) wait(ID NodeID, code packetCode, handleRes waitIsDone) error {
	errch := make(chan error, 1)

	select {
	case d.wtl.add <- &wait{
		expectFrom: ID,
		expectCode: code,
		handle:     handleRes,
		expiration: time.Now().Add(watingTimeout),
		errch:      errch,
	}:
	case <-d.term:
		errch <- errStopped
	}

	return <-errch
}

func (d *udpAgent) stop() {
	discvLog.Info("discovery agent stop")

	select {
	case <-d.term:
	default:
		close(d.term)
	}
}

func (d *udpAgent) readLoop() {
	buf := make([]byte, maxPacketLength)

loop:
	for {
		select {
		case <-d.term:
			break loop
		default:
		}

		n, addr, err := d.conn.ReadFromUDP(buf)

		p, err := unPacket(buf[:n])
		if err != nil {
			discvLog.Error(fmt.Sprintf("unpack message from %s error: %v\n", addr, err))
			continue
		}
		p.from = addr

		err = d.packetHandler(p)
		if err != nil {
			discvLog.Error(fmt.Sprintf("handle message %s from %s@%s error: %v\n", p.code, p.fromID, p.from, err))
		}
	}

	d.conn.Close()
}

func (d *udpAgent) send(addr *net.UDPAddr, code packetCode, m Message) (hash types.Hash, err error) {
	data, hash, err := m.pack(d.priv)

	if err != nil {
		discvLog.Error(fmt.Sprintf("pack message %s to %s error: %v\n", code, addr, err))
		return
	}

	n, err := d.conn.WriteToUDP(data, addr)
	if err != nil {
		discvLog.Error(fmt.Sprintf("send message %s to %s error: %v\n", code, addr, err))
		return
	}

	if n != len(data) {
		err = fmt.Errorf("send incomplete message %s (%d/%dbytes) to %s\n", code, n, len(data))
		discvLog.Error(err.Error())
		return
	}

	discvLog.Info(fmt.Sprintf("send message %s to %s done\n", code, addr))

	return
}

func (d *udpAgent) need(p *packet) bool {
	matched := make(chan bool, 1)
	select {
	case <-d.term:
		return false
	case d.wtl.rec <- &res{
		from:    p.fromID,
		code:    p.code,
		data:    p.msg,
		matched: matched,
	}:
		return <-matched
	}
}
