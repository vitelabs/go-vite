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
const topoCmd = 2
const traceCmd = 3

const headerLength = 32
const maxPayloadSize = ^uint64(0)

var errMsgTooLarge = errors.New("message payload is two large")
var errPeerTermed = errors.New("peer has been terminated")
var errProtoHandleDone = errors.New("protocol has done")

type transport interface {
	MsgReadWriter
	close(err error)
	Handshake(ours *Handshake) (their *Handshake, err error)
}

type conn struct {
	fd net.Conn
	transport
	flags   connFlag
	term    chan struct{}
	id      discovery.NodeID
	cmdSets []*CmdSet
	name    string
}

func (c *conn) is(flag connFlag) bool {
	return c.flags.is(flag)
}

type protoFrame struct {
	*Protocol
	input    chan *Msg
	term     chan struct{}
	canWrite chan struct{} // keep multiple protoFrames of the same peer writeMsg synchronously
	writeErr chan error    // indicate write done, and whether there is an error
	w        MsgWriter
}

func newProtoFrame(protocol *Protocol, rw MsgReadWriter) *protoFrame {
	return &protoFrame{
		Protocol: protocol,
		input:    make(chan *Msg, 1),
		w:        rw,
	}
}

func (pf *protoFrame) ReadMsg() (msg Msg, err error) {
	select {
	case <-pf.term:
		return msg, io.EOF
	case m := <-pf.input:
		return *m, nil
	}
}

func (pf *protoFrame) WriteMsg(msg Msg) error {
	if msg.CmdSetID != pf.ID {
		return fmt.Errorf("protoFrame %x cannot write message of CmdSet %x", pf.ID, msg.CmdSetID)
	}

	select {
	case <-pf.term:
		return errPeerTermed
	case <-pf.canWrite:
		err := pf.w.WriteMsg(msg)
		pf.writeErr <- err
	}

	return nil
}

// @section Peer
type Peer struct {
	rw          *conn
	protoFrames map[string]*protoFrame
	created     time.Time
	wg          sync.WaitGroup
	term        chan struct{}
	disc        chan DiscReason
	protoErr    chan error
	log         log15.Logger
}

func NewPeer(conn *conn, ourSet []*Protocol) (*Peer, error) {
	protoFrames := createProtoFrames(ourSet, conn.cmdSets, conn)

	if len(protoFrames) == 0 {
		return nil, DiscUselessPeer
	}

	return &Peer{
		rw:          conn,
		protoFrames: protoFrames,
		created:     time.Now(),
		term:        make(chan struct{}),
		disc:        make(chan DiscReason),
		log:         log15.New("module", "p2p/peer"),
		protoErr:    make(chan error, len(protoFrames)+1), // additional baseProtocol error
	}, nil
}

// create multiple protoFrames above the rw
func createProtoFrames(ourSet []*Protocol, theirSet []*CmdSet, rw MsgReadWriter) map[string]*protoFrame {
	protoFrames := make(map[string]*protoFrame)
	for _, our := range ourSet {
		for _, their := range theirSet {
			if our.ID == their.ID && our.Name == their.Name {
				protoFrames[our.String()] = newProtoFrame(our, rw)
			}
		}
	}

	return protoFrames
}

func (p *Peer) ID() discovery.NodeID {
	return p.rw.id
}

func (p *Peer) Name() string {
	return p.rw.name
}

func (p *Peer) String() string {
	return p.ID().String() + "@" + p.RemoteAddr().String()
}

func (p *Peer) CmdSets() []*CmdSet {
	return p.rw.cmdSets
}

func (p *Peer) RemoteAddr() net.Addr {
	return p.rw.fd.RemoteAddr()
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
		Inbound: p.rw.is(inbound),
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
		if pf.CmdSet().ID == CmdSetID {
			return pf
		}
	}

	return nil
}

func (p *Peer) runProtocols() (chan<- struct{}, <-chan error) {
	canWrite := make(chan struct{}, 1)
	writeErr := make(chan error, 1)

	p.wg.Add(len(p.protoFrames))

	for _, proto := range p.protoFrames {
		proto.term = p.term
		proto.canWrite = canWrite
		proto.writeErr = writeErr
		go func(pf *protoFrame) {
			defer p.wg.Done()
			err := pf.Handle(p, pf)
			if err == nil {
				err = errProtoHandleDone
			}

			p.protoErr <- err
		}(proto)
	}

	return canWrite, writeErr
}

func (p *Peer) run() (err error) {
	canWrite, writeErr := p.runProtocols()
	canWrite <- struct{}{}

	readErr := make(chan error, 1)

	p.wg.Add(1)
	go p.readLoop(readErr)

	var reason DiscReason

	for {
		select {
		case err = <-readErr:
			if r, ok := err.(DiscReason); ok {
				reason = r
			} else {
				reason = DiscNetworkError
			}
			goto END
		case err = <-writeErr:
			if err != nil {
				reason = DiscNetworkError
				goto END
			}
			canWrite <- struct{}{}
		case err = <-p.protoErr:
			reason = errTodiscReason(err)
			goto END
		case reason = <-p.disc:
			reason = errTodiscReason(reason)
			goto END
		}
	}

END:
	close(p.term)
	p.wg.Wait()
	p.rw.close(reason)

	return err
}

func (p *Peer) readLoop(out chan<- error) {
	defer p.wg.Done()

	for {
		msg, err := p.rw.ReadMsg()
		if err != nil {
			p.log.Error("peer read error", "ID", p.ID().String(), "error", err)
			out <- err
			return
		}

		msg.ReceivedAt = time.Now()

		err = p.handleMsg(&msg)
		if err != nil {
			out <- err
			return
		}
	}
}

func (p *Peer) handleMsg(msg *Msg) error {
	p.log.Info("peer handle message", "CmdSet", msg.CmdSetID, "Cmd", msg.Cmd, "from", p.ID().String())

	cmdset, cmd := msg.CmdSetID, msg.Cmd

	if cmdset == baseProtocolCmdSet {
		switch cmd {
		case discCmd:
			reason, err := ReadDiscReason(msg.Payload)
			if err != nil {
				return err
			}
			return reason
		case topoCmd:
			// todo
		default:
			return msg.Discard()
		}
	} else {
		pf := p.protoFrame(cmdset)
		if pf == nil {
			return fmt.Errorf("missing suitable protoFrame to handle message %d/%d\n", cmdset, cmd)
		} else {
			select {
			case <-p.term:
				p.log.Error(fmt.Sprintf("peer has been terminated, cannot handle message %d/%d\n", cmdset, cmd))
				return errPeerTermed
			case pf.input <- msg:
			default:
				p.log.Warn(fmt.Sprintf("protoFrame is busy, discard message %d/%d\n", cmdset, cmd))
				return msg.Discard()
			}
		}
	}

	return nil
}

// @section PeerSet
type PeerSet struct {
	peers    map[discovery.NodeID]*Peer
	inbound  int
	outbound int
}

func newPeerSet() *PeerSet {
	return &PeerSet{
		peers: make(map[discovery.NodeID]*Peer),
	}
}

func (s *PeerSet) Add(p *Peer) error {
	if _, ok := s.peers[p.ID()]; ok {
		return DiscAlreadyConnected
	}

	s.peers[p.ID()] = p
	if p.rw.is(inbound) {
		s.inbound++
	} else {
		s.outbound++
	}

	return nil
}

func (s *PeerSet) Del(p *Peer) {
	delete(s.peers, p.ID())
	if p.rw.is(inbound) {
		s.inbound--
	} else {
		s.outbound--
	}
}

func (s *PeerSet) Has(id discovery.NodeID) bool {
	_, ok := s.peers[id]
	return ok
}

func (s *PeerSet) Clear(p *Peer) {
	s.peers = nil
	s.inbound = 0
	s.outbound = 0
}

func (s *PeerSet) Size() int {
	return len(s.peers)
}

func (s *PeerSet) Info() []*PeerInfo {
	info := make([]*PeerInfo, s.Size())
	i := 0
	for _, p := range s.peers {
		info[i] = p.Info()
		i++
	}

	return info
}

func (s *PeerSet) Traverse(fn func(id discovery.NodeID, p *Peer)) {
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
