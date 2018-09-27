package p2p

import (
	"encoding/binary"
	"fmt"
	"github.com/golang/snappy"
	"github.com/vitelabs/go-vite/log15"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

//var bufferPool = sync.Pool{
//	New: func() interface{} {
//		return new(bytes.Buffer)
//	},
//}
//
//func Buffer() *bytes.Buffer {
//	return bufferPool.Get().(*bytes.Buffer)
//}
//
//func EncodeToReader(s Serializable) (r io.Reader, size uint64, err error) {
//	data, err := s.Serialize()
//	if err != nil {
//		return
//	}
//
//	return BytesToReader(data), uint64(len(data)), nil
//}
//
//func BytesToReader(data []byte) io.Reader {
//	buf := bufferPool.Get().(*bytes.Buffer)
//	buf.Reset()
//
//	n, err := buf.Write(data)
//	if err != nil || n != len(data) {
//		bufferPool.Put(buf)
//		return bytes.NewReader(data)
//	}
//
//	return buf
//}

func PackMsg(cmdSetId, cmd, id uint64, s Serializable) (*Msg, error) {
	data, err := s.Serialize()
	if err != nil {
		return nil, err
	}

	return &Msg{
		CmdSetID:   cmdSetId,
		Cmd:        cmd,
		Id:         id,
		Size:       uint64(len(data)),
		Payload:    data,
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
	if _, err = writer.Write(head); err != nil {
		return
	}

	// write payload
	_, err = writer.Write(msg.Payload)

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

type AsyncMsgConn struct {
	fd    net.Conn
	rw    *MsgRw
	handler func(msg *Msg)
	term chan struct{}
	wg sync.WaitGroup
	_wqueue chan *Msg
	_rqueue chan *Msg
	_close error	// close reason
	log log15.Logger
	errored int32	// atomic, indicate whehter there is an error
	errch chan error	// report errch to upper layer
}

// create an AsyncMsgConn, fd is the basic connection
func NewAsyncMsgConn(fd net.Conn, handler func(msg *Msg)) *AsyncMsgConn {
	return &AsyncMsgConn{
		fd: fd,
		rw: &MsgRw{
			fd: fd,
			compressible: true,
		},
		handler: handler,
		term: make(chan struct{}),
		_wqueue: make(chan *Msg, 100),
		_rqueue: make(chan *Msg, 100),
		errch: make(chan error),
		log: log15.New("module", "p2p/AsyncMsgConn", "end", fd.RemoteAddr().String()),
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
	case <- c.term:
	default:
		c._close = err
		c.log.Error(fmt.Sprintf("close: %v", err))
		close(c.term)
	}

	c.wg.Wait()
}

func (c *AsyncMsgConn) report(err error) {
	if atomic.CompareAndSwapInt32(&c.errored,0, 1) {
		c.errch <- err
	}
}

func (c *AsyncMsgConn) readLoop() {
	defer c.wg.Done()
	defer close(c._rqueue)

	var tempDelay time.Duration
	var maxDelay = time.Second

	for {
		select {
		case <- c.term:
			return
		default:
		}

		c.fd.SetReadDeadline(time.Now().Add(msgReadTimeout))
		msg, err := c.rw.ReadMsg()

		if err == nil {
			select {
			case c._rqueue <- msg:
			default:
				msg.Discard()
				c.log.Error(fmt.Sprintf("can`t put message %s to read_queue<%d>, then discard", msg, len(c._rqueue)))
			}
		} else if err, ok := err.(net.Error); ok && err.Temporary() {
			if tempDelay == 0 {
				tempDelay = 5 * time.Millisecond
			} else {
				tempDelay *= 2
			}

			if tempDelay > maxDelay {
				tempDelay = maxDelay
			}

			c.log.Warn(fmt.Sprintf("udp read tempError, wait %d Millisecond", tempDelay))

			time.Sleep(tempDelay)
		} else {
			c.log.Error(fmt.Sprintf("read message error: %v", err))
			c.report(err)
			return
		}
	}
}

func (c *AsyncMsgConn) _write(msg *Msg) {
	// there is an error
	if atomic.LoadInt32(&c.errored) == 1 {
		return
	}

	c.fd.SetWriteDeadline(time.Now().Add(msgWriteTimeout))
	err := c.rw.WriteMsg(msg)
	if err != nil {
		c.report(err)
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
		case <- c.term:
			break loop
		case msg := <- c._wqueue:
			c._write(msg)
		}
	}

	close(c._wqueue)

	for msg := range c._wqueue {
		c._write(msg)
	}

	if reason, ok := c._close.(DiscReason); ok && reason != DiscNetworkError {
		c.Send(baseProtocolCmdSet, discCmd, 0, reason)
	}
}

func (c *AsyncMsgConn) handleLoop() {
	defer c.wg.Done()

	loop:
	for {
		select {
		case <- c.term:
			break loop
		case msg := <- c._rqueue:
			c.handler(msg)
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

func (c *AsyncMsgConn) Handshake(ours *Handshake) (their *Handshake, err error) {
	msg, err := PackMsg(baseProtocolCmdSet, handshakeCmd, 0, ours)
	if err != nil {
		return
	}

	errch := make(chan error, 1)
	go func() {
		errch <- c.rw.WriteMsg(msg)
	}()

	if their, err = readHandshake(c.rw); err != nil {
		return
	}

	if err = <- errch; err != nil {
		return
	}

	return
}

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
	err = h.Deserialize(msg.Payload)

	return
}

func Send(w MsgWriter, cmdset, cmd, id uint64, s Serializable) error {
	msg, err := PackMsg(cmdset, cmd, id, s)
	if err != nil {
		return err
	}

	return w.WriteMsg(msg)
}

func SendMsg(w MsgWriter, msg *Msg) error {
	return w.WriteMsg(msg)
}
