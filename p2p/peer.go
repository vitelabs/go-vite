package p2p

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"io"
	"net"
	"sync"
	"time"
)

const Version uint64 = 2
const compressibleVersion = 2

const baseProtocolCmdSet = 0
const maxBaseProtocolPayloadSize = 10 * 1024

const handshakeCmd = 0
const discCmd = 1

const headerLength = 32
const maxPayloadSize uint64 = ^uint64(0)>>32 - 1

const paralProtoFrame = 4

var errMsgTooLarge = errors.New("message payload is two large")
var errPeerTermed = errors.New("peer has been terminated")
var errProtoHandleDone = errors.New("all protocols done")
var errPeerTsBusy = errors.New("peer transport is busy, can`t write message")

type conn struct {
	*AsyncMsgConn
	flags   connFlag
	cmdSets []*CmdSet
	name    string
	id      discovery.NodeID
}

func (c *conn) is(flag connFlag) bool {
	return c.flags.is(flag)
}

type protoFrame struct {
	*Protocol
	*conn
	input    chan *Msg
	term     chan struct{}
	canWrite chan struct{} // if this frame can write message
}

func newProtoFrame(protocol *Protocol, conn *conn) *protoFrame {
	return &protoFrame{
		Protocol: protocol,
		conn:     conn,
		input:    make(chan *Msg, 10),
	}
}

func (pf *protoFrame) ReadMsg() (msg *Msg, err error) {
	select {
	case <-pf.term:
		return msg, io.EOF
	case m := <-pf.input:
		return m, nil
	}
}

func (pf *protoFrame) WriteMsg(msg *Msg) error {
	if msg.CmdSetID != pf.ID {
		return fmt.Errorf("protoFrame %x cannot write message of CmdSet %x", pf.ID, msg.CmdSetID)
	}

	select {
	case <-pf.term:
		return errPeerTermed
	case pf.canWrite <- struct{}{}:
		if pf.conn.SendMsg(msg) {
			<-pf.canWrite
			return nil
		}

		<-pf.canWrite
		return errPeerTsBusy
	}
}

type protoDone struct {
	name string
	err  error
}

// @section Peer
type Peer struct {
	ts          *conn
	protoFrames map[string]*protoFrame
	created     time.Time
	wg          sync.WaitGroup
	term        chan struct{}
	disc        chan DiscReason // for disconnect signal particularly
	errch       chan error      // for common error
	protoDone   chan *protoDone // for protocols
	log         log15.Logger
}

func NewPeer(conn *conn, ourSet []*Protocol) (*Peer, error) {
	protoFrames := createProtoFrames(ourSet, conn.cmdSets, conn)

	if len(protoFrames) == 0 {
		return nil, DiscUselessPeer
	}

	p := &Peer{
		ts:          conn,
		protoFrames: protoFrames,
		created:     time.Now(),
		term:        make(chan struct{}),
		disc:        make(chan DiscReason, 1),
		errch:       make(chan error, 1),
		log:         log15.New("module", "p2p/peer"),
		protoDone:   make(chan *protoDone, len(protoFrames)),
	}

	p.ts.handler = p.handleMsg

	return p, nil
}

// create multiple protoFrames above the rw
func createProtoFrames(ourSet []*Protocol, theirSet []*CmdSet, conn *conn) map[string]*protoFrame {
	protoFrames := make(map[string]*protoFrame)
	for _, our := range ourSet {
		for _, their := range theirSet {
			if our.ID == their.ID && our.Name == their.Name {
				protoFrames[our.String()] = newProtoFrame(our, conn)
			}
		}
	}

	return protoFrames
}

func (p *Peer) ID() discovery.NodeID {
	return p.ts.id
}

func (p *Peer) Name() string {
	return p.ts.name
}

func (p *Peer) String() string {
	return p.ID().String() + "@" + p.RemoteAddr().String()
}

func (p *Peer) CmdSets() []*CmdSet {
	return p.ts.cmdSets
}

func (p *Peer) RemoteAddr() *net.TCPAddr {
	return p.ts.fd.RemoteAddr().(*net.TCPAddr)
}

func (p *Peer) IP() net.IP {
	return p.RemoteAddr().IP
}

func (p *Peer) Info() *PeerInfo {
	cmdsets := p.CmdSets()
	cmdSetsInfo := make([]string, len(cmdsets))
	for i, cmdset := range cmdsets {
		cmdSetsInfo[i] = cmdset.String()
	}

	return &PeerInfo{
		ID:      p.ID().String(),
		Name:    p.Name(),
		CmdSets: cmdSetsInfo,
		Address: p.RemoteAddr().String(),
		Inbound: p.ts.is(inbound),
	}
}

func (p *Peer) Disconnect(reason DiscReason) {
	p.log.Info("disconnected", "peer", p.String(), "reason", reason)

	select {
	case <-p.term:
	case p.disc <- reason:
	}
}

func (p *Peer) protoFrame(CmdSetID uint64) *protoFrame {
	for _, pf := range p.protoFrames {
		if pf.ID == CmdSetID {
			return pf
		}
	}

	return nil
}

func (p *Peer) runProtocols() {
	p.wg.Add(len(p.protoFrames))
	canWrite := make(chan struct{}, paralProtoFrame)

	for _, proto := range p.protoFrames {
		go p.runProtocol(proto, canWrite)
	}
}

func (p *Peer) runProtocol(proto *protoFrame, canWrite chan struct{}) {
	defer p.wg.Done()
	proto.term = p.term
	proto.canWrite = canWrite

	err := proto.Handle(p, proto)
	p.protoDone <- &protoDone{proto.String(), err}
}

func (p *Peer) run() (err error) {
	p.log.Info(fmt.Sprintf("peer %s run", p))

	p.ts.Start()

	p.runProtocols()

loop:
	for {
		select {
		case err = <-p.disc:
			// we have been told will disconnect
			break loop
		case e := <-p.protoDone:
			p.log.Error(fmt.Sprintf("protocol %s is done: %v", e.name, err))
			delete(p.protoFrames, e.name)
			if len(p.protoFrames) == 0 {
				// all protocols have done
				err = errProtoHandleDone
				break loop
			}
		case err = <-p.ts.errch: // error occur
			break loop
		case err = <-p.errch:
			break loop
		}
	}

	close(p.term)
	p.ts.Close(err)
	p.wg.Wait()

	p.log.Info(fmt.Sprintf("peer %s run done: %v", p, err))
	return err
}

func (p *Peer) handleMsg(msg *Msg) {
	p.log.Info("peer handle message", "CmdSet", msg.CmdSetID, "Cmd", msg.Cmd, "from", p.ID().String())

	cmdset, cmd := msg.CmdSetID, msg.Cmd

	if cmdset == baseProtocolCmdSet {
		switch cmd {
		case discCmd:
			reason, err := DeserializeDiscReason(msg.Payload)
			if err == nil {
				p.disc <- reason
			}
			p.errch <- err
		default:
			msg.Discard()
		}
	} else {
		pf := p.protoFrame(cmdset)
		if pf == nil {
			// may be error occur concurrently
			select {
			case p.errch <- fmt.Errorf("missing suitable protoFrame to handle message %d/%d", cmdset, cmd):
			default:
			}
		} else {
			select {
			case <-p.term:
				p.log.Error(fmt.Sprintf("peer has been terminated, cannot handle message %d/%d", cmdset, cmd))
			case pf.input <- msg:
			default:
				p.log.Warn(fmt.Sprintf("protoFrame is busy, discard message %d/%d", cmdset, cmd))
				msg.Discard()
			}
		}
	}
}

// @section PeerSet
type PeerSet struct {
	peers    map[discovery.NodeID]*Peer
	lock     sync.RWMutex
	inbound  int
	outbound int
}

func NewPeerSet() *PeerSet {
	return &PeerSet{
		peers: make(map[discovery.NodeID]*Peer),
	}
}

func (s *PeerSet) Add(p *Peer) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.peers[p.ID()]; ok {
		return DiscAlreadyConnected
	}

	s.peers[p.ID()] = p
	if p.ts.is(inbound) {
		s.inbound++
	} else {
		s.outbound++
	}

	return nil
}

func (s *PeerSet) Del(p *Peer) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.peers, p.ID())
	if p.ts.is(inbound) {
		s.inbound--
	} else {
		s.outbound--
	}
}

func (s *PeerSet) Has(id discovery.NodeID) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	_, ok := s.peers[id]
	return ok
}

func (s *PeerSet) Clear(p *Peer) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.peers = nil
	s.inbound = 0
	s.outbound = 0
}

func (s *PeerSet) Size() int {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return len(s.peers)
}

func (s *PeerSet) Info() []*PeerInfo {
	s.lock.RLock()
	defer s.lock.RUnlock()

	info := make([]*PeerInfo, s.Size())
	i := 0
	for _, p := range s.peers {
		info[i] = p.Info()
		i++
	}

	return info
}

func (s *PeerSet) Traverse(fn func(id discovery.NodeID, p *Peer)) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	for id, p := range s.peers {
		id, p := id, p
		fn(id, p)
	}
}

// @section PeerInfo
type PeerInfo struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	CmdSets []string `json:"caps"`
	Address string   `json:"address"`
	Inbound bool     `json:"inbound"`
}
