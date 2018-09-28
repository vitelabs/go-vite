package p2p

import (
	"encoding/binary"
	"fmt"
	"github.com/golang/snappy"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

func PackMsg(cmdSetId, cmd, id uint64, s Serializable) (*Msg, error) {
	data, err := s.Serialize()
	if err != nil {
		return nil, err
	}

	return &Msg{
		CmdSetID: cmdSetId,
		Cmd:      cmd,
		Id:       id,
		Size:     uint64(len(data)),
		Payload:  data,
	}, nil
}

func ReadMsg(reader io.Reader, compressible bool) (msg *Msg, err error) {
	head := make([]byte, headerLength)
	if _, err = io.ReadFull(reader, head); err != nil {
		return
	}

	msg = new(Msg)
	msg.CmdSetID = binary.BigEndian.Uint64(head[:8])
	msg.Cmd = binary.BigEndian.Uint64(head[8:16])
	msg.Id = binary.BigEndian.Uint64(head[16:24])
	msg.Size = binary.BigEndian.Uint64(head[24:32])

	if msg.Size > maxPayloadSize {
		err = errMsgTooLarge
		return
	}

	payload := make([]byte, msg.Size)
	n, err := io.ReadFull(reader, payload)
	if err != nil {
		return
	}
	if uint64(n) != msg.Size {
		err = fmt.Errorf("read incomplete message %d/%d", n, msg.Size)
		return
	}

	if compressible {
		var fullSize int
		fullSize, err = snappy.DecodedLen(payload)
		if err != nil {
			return
		}

		payload, err = snappy.Decode(nil, payload)
		if err != nil {
			return
		}
		msg.Size = uint64(fullSize)
	}

	msg.Payload = payload
	msg.ReceivedAt = time.Now()

	return
}

func WriteMsg(writer io.Writer, compressible bool, msg *Msg) (err error) {
	if msg.Size > maxPayloadSize {
		return errMsgTooLarge
	}

	if compressible {
		msg.Payload = snappy.Encode(nil, msg.Payload)
		msg.Size = uint64(len(msg.Payload))
	}

	head := make([]byte, headerLength)
	binary.BigEndian.PutUint64(head[:8], msg.CmdSetID)
	binary.BigEndian.PutUint64(head[8:16], msg.Cmd)
	binary.BigEndian.PutUint64(head[16:24], msg.Id)
	binary.BigEndian.PutUint64(head[24:32], msg.Size)

	// write header
	var n int
	if n, err = writer.Write(head); err != nil {
		return
	} else if n != headerLength {
		return fmt.Errorf("write incomplement message header %d/%d bytes", n, headerLength)
	}

	// write payload
	if n, err = writer.Write(msg.Payload); err != nil {
		return
	} else if uint64(n) != msg.Size {
		return fmt.Errorf("write incomplement message %d/%d bytes", n, msg.Size)
	}

	return
}

// the most basic Msg reader & writer, Thread unsafe
type MsgRw struct {
	fd           io.ReadWriter
	compressible bool
}

func (rw *MsgRw) ReadMsg() (msg *Msg, err error) {
	return ReadMsg(rw.fd, rw.compressible)
}

func (rw *MsgRw) WriteMsg(msg *Msg) (err error) {
	return WriteMsg(rw.fd, rw.compressible, msg)
}

// AsyncMsgConn
var handshakeTimeout = 10 * time.Second
var msgReadTimeout = 40 * time.Second
var msgWriteTimeout = 20 * time.Second

const readBufferLen = 100
const writeBufferLen = 100

type AsyncMsgConn struct {
	fd      net.Conn
	rw      *MsgRw
	handler func(msg *Msg)
	term    chan struct{}
	wg      sync.WaitGroup
	_wqueue chan *Msg
	_rqueue chan *Msg
	_reason error // close reason
	log     log15.Logger
	errored int32      // atomic, indicate whehter there is an error, readErr(1), writeErr(2)
	errch   chan error // report errch to upper layer
}

// create an AsyncMsgConn, fd is the basic connection
func NewAsyncMsgConn(fd net.Conn, handler func(msg *Msg)) *AsyncMsgConn {
	return &AsyncMsgConn{
		fd: fd,
		rw: &MsgRw{
			fd:           fd,
			compressible: true,
		},
		handler: handler,
		term:    make(chan struct{}),
		_wqueue: make(chan *Msg, writeBufferLen),
		_rqueue: make(chan *Msg, readBufferLen),
		errch:   make(chan error, 1),
		log:     log15.New("module", "p2p/AsyncMsgConn", "end", fd.RemoteAddr().String()),
	}
}

func (c *AsyncMsgConn) Start() {
	c.wg.Add(1)
	go c.readLoop()

	c.wg.Add(1)
	go c.writeLoop()

	c.wg.Add(1)
	go c.handleLoop()
}

func (c *AsyncMsgConn) Close(err error) {
	select {
	case <-c.term:
	default:
		c.log.Error(fmt.Sprintf("close: %v", err))
		c._reason = err
		close(c.term)
		c.wg.Wait()
		c.log.Error(fmt.Sprintf("closed: %v", err))
	}
}

func (c *AsyncMsgConn) report(t int32, err error) {
	if atomic.CompareAndSwapInt32(&c.errored, 0, t) {
		c.errch <- err
	}
}

func (c *AsyncMsgConn) readLoop() {
	defer c.wg.Done()
	defer close(c._rqueue)

	for {
		select {
		case <-c.term:
			return
		default:
		}

		msg, err := c.rw.ReadMsg()

		if err == nil {
			c._rqueue <- msg
			monitor.LogEvent("async-conn-read", c.fd.RemoteAddr().String())
		} else {
			c.log.Error(fmt.Sprintf("read message error: %v", err))
			c.report(1, err)
			return
		}
	}
}

func (c *AsyncMsgConn) _write(msg *Msg) {
	// there is an error
	if atomic.LoadInt32(&c.errored) != 0 {
		return
	}

	err := c.rw.WriteMsg(msg)
	monitor.LogEvent("async-conn-write", c.fd.RemoteAddr().String())

	if err != nil {
		c.report(2, err)
		c.log.Error(fmt.Sprintf("write message %s to %s error: %v", msg, c.fd.RemoteAddr(), err))
	} else {
		c.log.Info(fmt.Sprintf("write message %s to %s done", msg, c.fd.RemoteAddr()))
	}
}

func (c *AsyncMsgConn) writeLoop() {
	defer c.wg.Done()
	defer c.fd.Close()

loop:
	for {
		select {
		case <-c.term:
			break loop
		case msg := <-c._wqueue:
			c._write(msg)
		}
	}

	close(c._wqueue)

	if atomic.LoadInt32(&c.errored) == 0 {
		for msg := range c._wqueue {
			c._write(msg)
		}

		if reason, ok := c._reason.(DiscReason); ok && reason != DiscNetworkError {
			c.Send(baseProtocolCmdSet, discCmd, 0, reason)
		}
	}
}

func (c *AsyncMsgConn) handleLoop() {
	defer c.wg.Done()

loop:
	for {
		select {
		case <-c.term:
			break loop
		case msg := <-c._rqueue:
			if msg != nil {
				c.handler(msg)
			} else {
				break loop
			}
		}
	}

	for msg := range c._rqueue {
		c.handler(msg)
	}
}

// send a message asynchronously, put message into a internal buffered channel before send it
// if the internal channel is full, return false
func (c *AsyncMsgConn) SendMsg(msg *Msg) bool {
	select {
	case <-c.term:
		return false
	case c._wqueue <- msg:
		return true
	default:
		return false
	}
}

func (c *AsyncMsgConn) Send(cmdset, cmd, id uint64, s Serializable) bool {
	msg, err := PackMsg(cmdset, cmd, id, s)

	if err != nil {
		return false
	}

	return c.SendMsg(msg)
}

func (c *AsyncMsgConn) encHandshake() {
	// todo
}

// send Handshake data, after signature with ed25519 algorithm
func (c *AsyncMsgConn) Handshake(data []byte) (their *Handshake, err error) {
	send := make(chan error, 1)
	go sendHandMsg(c.rw, &Msg{
		CmdSetID: baseProtocolCmdSet,
		Cmd:      handshakeCmd,
		Id:       0,
		Size:     uint64(len(data)),
		Payload:  data,
	}, send)

	if their, err = readHandshake(c.rw); err != nil {
		return
	}

	if err = <-send; err != nil {
		return
	}

	return
}

func sendHandMsg(w MsgWriter, hand *Msg, sendErr chan<- error) {
	sendErr <- w.WriteMsg(hand)
}

var errorHandshakeVerify = errors.New("signature of handshake Msg verify failed")

func readHandshake(r MsgReader) (h *Handshake, err error) {
	msg, err := r.ReadMsg()
	if err != nil {
		return nil, err
	}
	if msg.CmdSetID != baseProtocolCmdSet {
		return nil, fmt.Errorf("should be baseProtocolCmdSet, got %x", msg.CmdSetID)
	}

	if msg.Cmd == discCmd {
		discReason, err := DeserializeDiscReason(msg.Payload)
		if err != nil {
			return nil, fmt.Errorf("disconnected, but parse DiscReason error: %v", err)
		}

		return nil, discReason
	}

	if msg.Cmd != handshakeCmd {
		return nil, fmt.Errorf("should be handshake message, but got %x", err)
	}

	h = new(Handshake)
	err = h.Deserialize(msg.Payload[64:])
	if err != nil {
		return
	}

	if !ed25519.Verify(h.ID[:], msg.Payload[64:], msg.Payload[:64]) {
		return nil, errorHandshakeVerify
	}

	return
}
