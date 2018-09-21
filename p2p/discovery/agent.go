package discovery

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/monitor"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const maxNeighborsOneTrip = 10

var watingTimeout = 10 * time.Second // must be enough little, at least than checkInterval

var errStopped = errors.New("discovery server has stopped")
var errWaitOvertime = errors.New("wait for response timeout")
var errSendToSelf = errors.New("send discovery message to self")

// after send query. wating for reply.
type waitIsDone func(Message) bool

type wait struct {
	expectFrom NodeID
	expectCode packetCode
	handle     waitIsDone
	expiration time.Time
	errch      chan error
	next       *wait
}

func (n *wait) traverse(fn func(prev, current *wait)) {
	prev, current := n, n.next

	for current != nil {
		fn(prev, current)
		prev, current = current, current.next
	}
}

type res struct {
	from    NodeID
	code    packetCode
	data    Message
	matched chan bool
}

type wtList struct {
	list  *wait
	count int
	add   chan *wait
	rec   chan *res
}

func newWtList() *wtList {
	return &wtList{
		list: new(wait),
		add:  make(chan *wait),
		rec:  make(chan *res),
	}
}

func (wtl *wtList) loop(stop <-chan struct{}) {
	checkTicker := time.NewTicker(watingTimeout)
	defer checkTicker.Stop()

	for {
		select {
		case <-stop:
			wtl.list.traverse(func(_, wt *wait) {
				wt.errch <- errStopped
			})
			return
		case w := <-wtl.add:
			wtl.list.next = w
			wtl.count++
		case r := <-wtl.rec:
			wtl.handle(r)
		case <-checkTicker.C:
			wtl.clean()
		}
	}
}

func (wtl *wtList) handle(rs *res) {
	matched := false
	wtl.list.traverse(func(prev, current *wait) {
		if current.expectFrom == rs.from && current.expectCode == rs.code {
			matched = true

			if current.handle(rs.data) {
				current.errch <- nil
				// remove current wait from list
				prev.next = current.next
				wtl.count--
			}
		}
	})

	rs.matched <- matched
}

func (wtl *wtList) clean() {
	now := time.Now()
	wtl.list.traverse(func(prev, current *wait) {
		if current.expiration.Before(now) {
			current.errch <- errWaitOvertime
			// remove current wait from list
			prev.next = current.next
			wtl.count--
		}
	})
}

// @section agent
type agent struct {
	self       *Node
	conn       *net.UDPConn
	priv       ed25519.PrivateKey
	term       chan struct{}
	running    int32 // atomic
	pktHandler func(*packet) error
	wtl        *wtList
	wg         sync.WaitGroup
}

// should run as goroutine
func (d *agent) start() {
	if !atomic.CompareAndSwapInt32(&d.running, 0, 1) {
		return
	}

	discvLog.Info("discovery agent start")

	d.wg.Add(1)
	go func() {
		d.wtl.loop(d.term)
		d.wg.Done()
	}()

	// should not run as goroutine
	d.wg.Add(1)
	go d.readLoop()
}

func (d *agent) stop() {
	if !atomic.CompareAndSwapInt32(&d.running, 1, 0) {
		return
	}

	discvLog.Info("discovery agent stop")

	close(d.term)
	d.wg.Wait()
}

// implements discvAgent interface
func (d *agent) ping(node *Node) error {
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
		discvLog.Error(fmt.Sprintf("send ping to %s error: %v", node, err))
		return err
	} else {
		discvLog.Info(fmt.Sprintf("send ping to %s done", node))
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

func (d *agent) pong(node *Node, ack types.Hash) error {
	if node.ID == d.self.ID {
		return nil
	}

	_, err := d.send(node.UDPAddr(), pongCode, &Pong{
		ID:         d.self.ID,
		Ping:       ack,
		Expiration: getExpiration(),
	})

	if err != nil {
		discvLog.Error(fmt.Sprintf("send ping to %s error: %v", node, err))
	} else {
		discvLog.Info(fmt.Sprintf("send ping to %s done", node))
	}

	return err
}

// should ping-pong checked before
func (d *agent) findnode(n *Node, ID NodeID) (nodes []*Node, err error) {
	monitor.LogEvent("p2p/discv", "findnode")

	_, err = d.send(n.UDPAddr(), findnodeCode, &FindNode{
		ID:         d.self.ID,
		Target:     ID,
		Expiration: getExpiration(),
	})

	if err != nil {
		discvLog.Error(fmt.Sprintf("send findnode %s to %s error: %v", ID, n, err))
		return
	} else {
		discvLog.Info(fmt.Sprintf("send findnode %s to %s done", ID, n))
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

func (d *agent) sendNeighbors(n *Node, nodes []*Node) (err error) {
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
			_, err = d.send(n.UDPAddr(), neighborsCode, neighbors)

			if err != nil {
				discvLog.Error(fmt.Sprintf("send %d neighbors to %s error: %v", len(carriage), n, err))
			} else {
				sent = true
				discvLog.Error(fmt.Sprintf("send %d neighbors to %s done", len(carriage), n))
			}
			carriage = carriage[:0]
		}
	}

	// send nodes even if the list is empty
	if !sent || len(carriage) > 0 {
		neighbors.Nodes = carriage
		_, err = d.send(n.UDPAddr(), neighborsCode, neighbors)

		if err != nil {
			discvLog.Error(fmt.Sprintf("send %d neighbors to %s error: %v", len(carriage), n, err))
		} else {
			discvLog.Info(fmt.Sprintf("send %d neighbors to %s done", len(carriage), n))
		}
	}

	return
}

func (d *agent) wait(ID NodeID, code packetCode, handle waitIsDone) error {
	errch := make(chan error, 1)

	select {
	case d.wtl.add <- &wait{
		expectFrom: ID,
		expectCode: code,
		handle:     handle,
		expiration: time.Now().Add(watingTimeout),
		errch:      errch,
	}:
	case <-d.term:
		errch <- errStopped
	}

	return <-errch
}

func (d *agent) readLoop() {
	defer d.wg.Done()
	defer d.conn.Close()

	buf := make([]byte, maxPacketLength)

	var tempDelay time.Duration
	var maxDelay = time.Second

	for {
		select {
		case <-d.term:
			return
		default:
			n, addr, err := d.conn.ReadFromUDP(buf)

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

					discvLog.Info(fmt.Sprintf("udp read tempError, wait %d Millisecond", tempDelay))

					time.Sleep(tempDelay)

					continue
				}

				return
			}

			tempDelay = 0

			p, err := unPacket(buf[:n])
			if err != nil {
				discvLog.Error(fmt.Sprintf("unpack message from %s error: %v", addr, err))
				go d.send(addr, exceptionCode, &Exception{
					Code: eCannotUnpack,
				})
				continue
			}

			monitor.LogEvent("p2p/discv", "msg")

			p.from = addr

			go d.pktHandler(p)
		}
	}
}

func (d *agent) send(addr *net.UDPAddr, code packetCode, m Message) (hash types.Hash, err error) {
	data, hash, err := m.pack(d.priv)

	if err != nil {
		discvLog.Error(fmt.Sprintf("pack message %s to %s error: %v", code, addr, err))
		return
	}

	n, err := d.conn.WriteToUDP(data, addr)
	if err != nil {
		discvLog.Error(fmt.Sprintf("send message %s to %s error: %v", code, addr, err))
		return
	}

	if n != len(data) {
		err = fmt.Errorf("send incomplete message %s (%d/%dbytes) to %s\n", code, n, len(data), addr)
		discvLog.Error(err.Error())
		return
	}

	monitor.LogEvent("p2p/discv", code.String())

	return
}

func (d *agent) need(p *packet) bool {
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
