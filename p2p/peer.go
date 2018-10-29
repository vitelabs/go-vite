package p2p

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"github.com/vitelabs/go-vite/p2p/protos"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const writeBufferLen = 100
const readBufferLen = 100

var msgReadTimeout = 40 * time.Second
var msgWriteTimeout = 20 * time.Second

var errMsgTooLarge = errors.New("message payload is too large")
var errMsgNull = errors.New("message payload is 0 byte")
var errPeerTermed = errors.New("peer has been terminated")

type transport struct {
	net.Conn
	flags      connFlag
	cmdSets    []CmdSet
	name       string
	localID    discovery.NodeID
	localIP    net.IP
	localPort  uint16
	remoteID   discovery.NodeID
	remoteIP   net.IP
	remotePort uint16
}

func (t *transport) ReadMsg() (*Msg, error) {
	t.SetReadDeadline(time.Now().Add(msgReadTimeout))
	return ReadMsg(t)
}

func (t *transport) WriteMsg(msg *Msg) error {
	t.SetWriteDeadline(time.Now().Add(msgWriteTimeout))
	return WriteMsg(t, msg)
}

func (t *transport) is(flag connFlag) bool {
	return t.flags.is(flag)
}

func (t *transport) Handshake(key ed25519.PrivateKey, our *Handshake) (their *Handshake, err error) {
	data, err := our.Serialize()
	if err != nil {
		return
	}
	sig := ed25519.Sign(key, data)
	// unshift signature before data
	data = append(sig, data...)

	send := make(chan error, 1)
	common.Go(func() {
		msg := NewMsg()
		msg.CmdSet = baseProtocolCmdSet
		msg.Cmd = handshakeCmd
		msg.Payload = data

		send <- WriteMsg(t, msg)
	})

	if their, err = readHandshake(t); err != nil {
		return
	}

	if err = <-send; err != nil {
		return
	}

	return
}

type ProtoFrame struct {
	*Protocol
	r         chan *Msg
	w         chan<- *Msg   // use peer`s wqueue
	term      chan struct{} // use peer`s term
	Received  map[Cmd]uint64
	Discarded map[Cmd]uint64
	Send      map[Cmd]uint64
}

func newProtoFrame(protocol *Protocol) *ProtoFrame {
	return &ProtoFrame{
		Protocol:  protocol,
		r:         make(chan *Msg, readBufferLen),
		Received:  make(map[Cmd]uint64),
		Discarded: make(map[Cmd]uint64),
		Send:      make(map[Cmd]uint64),
	}
}

func (pf *ProtoFrame) ReadMsg() (msg *Msg, err error) {
	select {
	case <-pf.term:
		return msg, io.EOF
	case msg = <-pf.r:
		return
	}
}

func (pf *ProtoFrame) WriteMsg(msg *Msg) (err error) {
	select {
	case <-pf.term:
		return errPeerTermed
	case pf.w <- msg:
		return
	}
}

// create multiple pfs above the rw
func createProtoFrames(ourSet []*Protocol, theirSet []CmdSet) pfMap {
	pfs := make(pfMap)
	for _, our := range ourSet {
		for _, their := range theirSet {
			if our.ID == their {
				pfs[our.ID] = newProtoFrame(our)
			}
		}
	}

	return pfs
}

// event
type protoDone struct {
	id  CmdSet
	err error
}

// @section Peer
type pfMap = map[CmdSet]*ProtoFrame
type Peer struct {
	ts        *transport
	tsError   int32 // atomic if transport occur an error
	pfs       pfMap
	Created   time.Time
	wg        sync.WaitGroup
	term      chan struct{}
	disc      chan DiscReason // disconnect proactively
	errch     chan error      // for common error
	protoDone chan *protoDone // for protocols
	wqueue    chan *Msg
	speed     float64
	rwLock    sync.RWMutex
	log       log15.Logger
}

func NewPeer(conn *transport, ourSet []*Protocol) (*Peer, error) {
	pfs := createProtoFrames(ourSet, conn.cmdSets)

	if len(pfs) == 0 {
		return nil, DiscUselessPeer
	}

	p := &Peer{
		ts:        conn,
		pfs:       pfs,
		term:      make(chan struct{}),
		Created:   time.Now(),
		disc:      make(chan DiscReason, 1),
		errch:     make(chan error),
		protoDone: make(chan *protoDone, len(pfs)),
		wqueue:    make(chan *Msg, writeBufferLen),
		log:       log15.New("module", "p2p/peer"),
	}

	return p, nil
}

func (p *Peer) Disconnect(reason DiscReason) {
	select {
	case <-p.term:
	case p.disc <- reason:
	}
}

func (p *Peer) startProtocols() {
	p.wg.Add(len(p.pfs))

	for _, pf := range p.pfs {
		// closure
		pf := pf
		common.Go(func() {
			defer p.wg.Done()

			//p.runProtocol(protoFrame)
			pf.term = p.term
			pf.w = p.wqueue

			err := pf.Handle(p, pf)
			p.protoDone <- &protoDone{pf.ID, err}
		})
	}
}

func (p *Peer) readLoop() {
	defer p.wg.Done()

	for {
		select {
		case <-p.term:
			return
		default:
			if msg, err := p.ts.ReadMsg(); err == nil {
				monitor.LogEvent("p2p_ts", "read")
				monitor.LogDuration("p2p_ts", "read_bytes", int64(len(msg.Payload)))
				monitor.LogDuration("p2p_ts", "stt", msg.ReceivedAt.Sub(msg.SendAt).Nanoseconds())

				p.handleMsg(msg)
			} else {
				select {
				case p.errch <- err:
					atomic.StoreInt32(&p.tsError, 1)
				default:
					return
				}
			}
		}
	}
}

func (p *Peer) writeLoop() {
	defer p.wg.Done()

loop:
	for {
		select {
		case <-p.term:
			break loop
		case msg := <-p.wqueue:
			if pf, ok := p.pfs[msg.CmdSet]; ok {
				pf.Send[msg.Cmd]++
			}

			before := time.Now()
			if err := p.ts.WriteMsg(msg); err != nil {
				select {
				case p.errch <- err:
					atomic.StoreInt32(&p.tsError, 2)
				default:
					return
				}
			}

			monitor.LogEvent("p2p_ts", "write")
			monitor.LogDuration("p2p_ts", "write_bytes", int64(len(msg.Payload)))
			monitor.LogDuration("p2p_ts", "write_queue", int64(len(p.wqueue)))
			monitor.LogTime("p2p_ts", "write_time", before)
			p.log.Debug(fmt.Sprintf("write message %d/%d to %s, spend %s, rest %d messages", msg.CmdSet, msg.Cmd, p.RemoteAddr(), time.Now().Sub(before), len(p.wqueue)))
		}
	}

	// no error, disconnected initiative
	if atomic.LoadInt32(&p.tsError) == 0 {
		for i := 0; i < len(p.wqueue); i++ {
			if err := p.ts.WriteMsg(<-p.wqueue); err != nil {
				return
			}
		}
	}
}

func (p *Peer) run() (err error) {
	p.log.Info(fmt.Sprintf("peer %s run", p))

	p.startProtocols()

	p.wg.Add(1)
	common.Go(p.readLoop)

	p.wg.Add(1)
	common.Go(p.writeLoop)

	var proactively bool // whether we want to disconnect or not
	var reason DiscReason

wait:
	select {
	case reason = <-p.disc:
		err = reason
		p.log.Warn(fmt.Sprintf("disconnect with peer %s: %v", p.RemoteAddr(), err))
		proactively = true

	case e := <-p.protoDone:
		if pf, ok := p.pfs[e.id]; ok {
			p.log.Error(fmt.Sprintf("peer %s protocol %s is done: %v", p.RemoteAddr(), pf, e.err))

			if err = e.err; err != nil {
				reason = DiscProtocolError
				proactively = true
				break
			}

			p.rwLock.Lock()
			delete(p.pfs, e.id)
			p.rwLock.Unlock()

			if len(p.pfs) == 0 {
				err = DiscAllProtocolDone
				reason = DiscAllProtocolDone
				proactively = true
				break
			}
		}

		// wait for rest protocols
		goto wait

	case err = <-p.errch:
		p.log.Error(fmt.Sprintf("peer %s error: %v", p.RemoteAddr(), err))
		proactively = false
	}

	if proactively && atomic.LoadInt32(&p.tsError) == 0 {
		if m, e := PackMsg(baseProtocolCmdSet, discCmd, 0, reason); e == nil {
			p.ts.WriteMsg(m)
		}
	}

	close(p.term)
	p.ts.Close()

	p.wg.Wait()

	p.log.Info(fmt.Sprintf("peer %s run done: %v", p.RemoteAddr(), err))
	return err
}

func (p *Peer) handleMsg(msg *Msg) {
	cmdset, cmd := msg.CmdSet, msg.Cmd

	if cmdset == baseProtocolCmdSet {
		if cmd == discCmd {
			p.log.Warn(fmt.Sprintf("receive disc from %s", p.RemoteAddr()))

			var disc error
			if reason, err := DeserializeDiscReason(msg.Payload); err == nil {
				disc = reason
			} else {
				disc = err
			}

			select {
			case <-p.term:
			case p.errch <- disc:
			default:
			}
		}

		msg.Recycle()
	} else if pf := p.pfs[cmdset]; pf != nil {
		select {
		case <-p.term:
			p.log.Error(fmt.Sprintf("peer has been terminated, can`t handle message %d/%d from %s", cmdset, cmd, p.RemoteAddr()))
			msg.Recycle()

		case pf.r <- msg:
			pf.Received[msg.Cmd]++
			monitor.LogDuration("p2p_ts", "read_queue_"+pf.String(), int64(len(pf.r)))
			p.log.Debug(fmt.Sprintf("read message %d/%d from %s, rest %d messages", msg.CmdSet, msg.Cmd, p.RemoteAddr(), len(pf.r)))

		default:
			p.log.Warn(fmt.Sprintf("protocol is busy, discard message %d/%d from %s", cmdset, cmd, p.RemoteAddr()))
			pf.Discarded[msg.Cmd]++
			monitor.LogEvent("p2p_ts", "discard")
			msg.Recycle()
		}
	} else {
		p.log.Error(fmt.Sprintf("missing suitable protocol to handle message %d/%d from %s", cmdset, cmd, p.RemoteAddr()))
		p.disc <- DiscUnKnownProtocol
	}
}

func (p *Peer) ID() discovery.NodeID {
	return p.ts.remoteID
}

func (p *Peer) Name() string {
	return p.ts.name
}

func (p *Peer) String() string {
	return p.ID().String() + "@" + p.RemoteAddr().String()
}

func (p *Peer) CmdSets() []CmdSet {
	return p.ts.cmdSets
}

func (p *Peer) RemoteAddr() *net.TCPAddr {
	return p.ts.RemoteAddr().(*net.TCPAddr)
}

func (p *Peer) IP() net.IP {
	return p.RemoteAddr().IP
}

func (p *Peer) Info() *PeerInfo {
	caps := make([]string, len(p.pfs))

	i := 0
	for _, pf := range p.pfs {
		caps[i] = pf.String()
		i++
	}

	return &PeerInfo{
		ID:      p.ID().String(),
		Name:    p.Name(),
		CmdSets: caps,
		Address: p.RemoteAddr().String(),
		Inbound: p.ts.is(inbound),
	}
}

// @section PeerSet
type PeerSet struct {
	peers    map[discovery.NodeID]*Peer
	inbound  int
	outbound int
	size     uint
}

func NewPeerSet() *PeerSet {
	return &PeerSet{
		peers: make(map[discovery.NodeID]*Peer),
	}
}

func (s *PeerSet) Add(p *Peer) {
	s.peers[p.ID()] = p
	if p.ts.is(inbound) {
		s.inbound++
	} else {
		s.outbound++
	}

	s.size++
}

func (s *PeerSet) Del(p *Peer) {
	delete(s.peers, p.ID())

	if p.ts.is(inbound) {
		s.inbound--
	} else {
		s.outbound--
	}

	s.size--
}

func (s *PeerSet) Has(id discovery.NodeID) bool {
	_, ok := s.peers[id]
	return ok
}

func (s *PeerSet) Size() uint {
	return s.size
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
	CmdSets []string `json:"cmdSets"`
	Address string   `json:"address"`
	Inbound bool     `json:"inbound"`
}

// @section ConnProperty
type ConnProperty struct {
	LocalID    string `json:"localID"`
	LocalIP    net.IP `json:"localIP"`
	LocalPort  uint16 `json:"localPort"`
	RemoteID   string `json:"remoteID"`
	RemoteIP   net.IP `json:"remoteIP"`
	RemotePort uint16 `json:"remotePort"`
}

func (cp *ConnProperty) Serialize() ([]byte, error) {
	return proto.Marshal(cp.Proto())
}

func (cp *ConnProperty) Deserialize(buf []byte) error {
	pb := new(protos.ConnProperty)
	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	cp.Deproto(pb)
	return nil
}

func (cp *ConnProperty) Proto() *protos.ConnProperty {
	return &protos.ConnProperty{
		LocalID:    cp.LocalID,
		LocalIP:    cp.LocalIP,
		LocalPort:  uint32(cp.LocalPort),
		RemoteID:   cp.RemoteID,
		RemoteIP:   cp.RemoteIP,
		RemotePort: uint32(cp.RemotePort),
	}
}

func (cp *ConnProperty) Deproto(pb *protos.ConnProperty) {
	cp.LocalID = pb.LocalID
	cp.LocalIP = pb.LocalIP
	cp.LocalPort = uint16(pb.LocalPort)

	cp.RemoteID = pb.RemoteID
	cp.RemoteIP = pb.RemoteIP
	cp.RemotePort = uint16(pb.RemotePort)
}

func (p *Peer) GetConnProperty() *ConnProperty {
	return &ConnProperty{
		LocalID:    p.ts.localID.String(),
		LocalIP:    p.ts.localIP,
		LocalPort:  p.ts.localPort,
		RemoteID:   p.ts.remoteID.String(),
		RemoteIP:   p.ts.remoteIP,
		RemotePort: p.ts.remotePort,
	}
}
