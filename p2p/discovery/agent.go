package discovery

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"net"
	"time"
)

var msgExpiration = 2 * time.Minute

// @section agent
var errStopped = errors.New("discv server has stopped")
var errTimeout = errors.New("waiting timeout")
var errUnmatchedPong = errors.New("unmatched pong hash")
var errUncheckedPacket = errors.New("msg before ping-pong checked")
var errMsgExpired = errors.New("receive expired answer msg")

type udpAgent struct {
	self    *Node
	conn    *net.UDPConn
	tab     *table
	priv    ed25519.PrivateKey
	waiting chan *wait
	res     chan *res
	stopped chan struct{}
}

// after send query. wating for reply.
type wait struct {
	expectFrom NodeID
	expectCode packetCode
	handleRes  func(Message) error
	expire     time.Time
	errch      chan error
}
type wtList []*wait

func handleRes(wtl wtList, rs *res) (rest wtList) {
	var err error
	for i, req := range wtl {
		if req.expectFrom == rs.from && req.expectCode == rs.code {
			if req.handleRes != nil {
				err = req.handleRes(rs.data)
			}

			rs.done <- err
			req.errch <- err

			rest = wtl[:i]
			rest = append(rest, wtl[i+1:]...)
			return wtl
		}
	}

	return wtl
}

func cleanStaleReq(rql wtList) wtList {
	now := time.Now()
	rest := make(wtList, 0, len(rql))
	for _, req := range rql {
		if req.expire.Before(now) {
			req.errch <- errTimeout
		} else {
			rest = append(rest, req)
		}
	}
	return rest
}

type res struct {
	from NodeID
	code packetCode
	data Message
	done chan error
}

func (d *udpAgent) ID() NodeID {
	return d.self.ID
}

// implements table.agent interface
func (d *udpAgent) ping(node *Node) error {
	if node.ID == d.ID() {
		return nil
	}

	discvLog.Info(fmt.Sprintf("ping %s\n", node))

	ping := &Ping{
		ID: d.ID(),
	}
	data, hash, err := ping.Pack(d.priv)
	if err != nil {
		return fmt.Errorf("pack ping msg error: %v\n", err)
	}

	_, err = d.conn.WriteToUDP(data, node.addr())
	if err != nil {
		return fmt.Errorf("ping %s error: %v\n", node, err)
	}

	sendTime := time.Now()
	node.lastping = time.Now()

	errch := d.wait(node.ID, pongCode, func(m Message) error {
		pong, ok := m.(*Pong)
		if ok {
			if pong.Ping == hash {
				if time.Now().Sub(sendTime) < msgExpiration {
					node.lastpong = time.Now()
					return nil
				}
				return errMsgExpired
			}
			return errUnmatchedPong
		}
		return errUncheckedPacket
	})

	return <-errch
}
func (d *udpAgent) findnode(ID NodeID, n *Node) (nodes []*Node, err error) {
	discvLog.Info(fmt.Sprintf("find %s to %s\n", ID, n))

	if time.Now().Sub(n.lastpong) > pingPongExpiration {
		err = d.ping(n)
		if err != nil {
			return nil, err
		}
	}

	err = d.send(n.addr(), findnodeCode, &FindNode{
		ID:     d.ID(),
		Target: ID,
	})
	if err != nil {
		return nodes, err
	}

	errch := d.wait(n.ID, neighborsCode, func(m Message) error {
		neighbors, ok := m.(*Neighbors)
		if !ok {
			return errors.New("receive unmatched msg, should be neighbors")
		}
		nodes = neighbors.Nodes
		return nil
	})

	err = <-errch

	if err != nil {
		discvLog.Error(fmt.Sprintf("find %s to %s error: %v\n", ID, n, err))
	} else {
		discvLog.Info(fmt.Sprintf("got %d neighbors of %s from %s\n", len(nodes), ID, n))
	}

	return nodes, err
}

func (d *udpAgent) wait(ID NodeID, code packetCode, handleRes func(Message) error) (errch chan error) {
	errch = make(chan error, 1)

	select {
	case d.waiting <- &wait{
		expectFrom: ID,
		expectCode: code,
		handleRes:  handleRes,
		expire:     time.Now().Add(watingTimeout),
		errch:      errch,
	}:
	case <-d.stopped:
		errch <- errStopped
	}

	return errch
}

func (d *udpAgent) close() {
	close(d.stopped)
	d.conn.Close()
}

func (d *udpAgent) loop() {
	var wtl wtList

	checkTicker := time.NewTicker(watingTimeout / 2)
	defer checkTicker.Stop()

	for {
		select {
		case <-d.stopped:
			for _, w := range wtl {
				w.errch <- errStopped
			}
		case req := <-d.waiting:
			wtl = append(wtl, req)
		case res := <-d.res:
			wtl = handleRes(wtl, res)
			discvLog.Info(fmt.Sprintf("handle msg %s from %s\n", res.code, res.from))
		case <-checkTicker.C:
			wtl = cleanStaleReq(wtl)
		}
	}
}

func (d *udpAgent) readLoop() {
	defer d.conn.Close()

	buf := make([]byte, maxPacketLength)

	for {
		n, addr, err := d.conn.ReadFromUDP(buf)

		if n == 0 {
			continue
		}

		m, hash, err := unPacket(buf[:n])
		if err != nil {
			discvLog.Error(fmt.Sprintf("unpack msg from %s error: %v\n", addr, err))
			continue
		}

		// todo
		// hash is just use for construct pong message,
		// could be optimize latter.
		err = m.Handle(d, addr, hash)
		if err != nil {
			discvLog.Error(fmt.Sprintf("handle msg from %s@%s error: %v\n", m.getID(), addr, err))
		} else {
			discvLog.Info(fmt.Sprintf("handle msg from %s@%s done\n", m.getID(), addr))
		}
	}
}

func (d *udpAgent) send(addr *net.UDPAddr, code packetCode, m Message) error {
	data, _, err := m.Pack(d.priv)

	if err != nil {
		discvLog.Error(fmt.Sprintf("send msg %s to %s error: %v\n", code, addr, err))
		return err
	}

	_, err = d.conn.WriteToUDP(data, addr)
	if err != nil {
		discvLog.Error(fmt.Sprintf("send msg %s to %s error: %v\n", code, addr, err))
		return err
	}

	discvLog.Info(fmt.Sprintf("send msg %s to %s done\n", code, addr))

	return nil
}

func (d *udpAgent) receive(code packetCode, m Message) error {
	done := make(chan error, 1)
	select {
	case <-d.stopped:
		return errStopped
	case d.res <- &res{
		from: m.getID(),
		code: code,
		data: m,
		done: done,
	}:
		return <-done
	}
}
