package net

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	net2 "net"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/vite/net/message"

	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p/vnode"
)

type syncCode byte

const (
	syncHandshake syncCode = iota
	syncHandshakeDone
	syncHandshakeErr
	syncRequest
	syncReady
	syncMissing
	syncNoAuth
	syncError
	syncQuit
)

type syncMsg struct {
	code    syncCode
	payload interface{}
}

type syncConnection interface {
	net2.Conn
	handshake() error
	download(from, to uint64) (err error)
	read() (syncMsg, error)
	write(syncMsg) error
}

type FileConnStatus struct {
	Id    string
	Addr  string
	Speed float64
}

type syncConn struct {
	net2.Conn
	id     vnode.NodeID
	busy   int32   // atomic
	t      int64   // timestamp
	speed  float64 // download speed, byte/s
	closed int32
	cacher syncCacher
	buf    []byte
	log    log15.Logger
}

func (f *syncConn) handshake() error {
	return nil
}

func (f *syncConn) read() (msg syncMsg, err error) {
	buf := make([]byte, 20)
	n, err := f.Conn.Read(buf)
	if err != nil {
		return
	}
	if n < 20 {
		err = errors.New("too short")
		return
	}

	msg.code = syncCode(buf[0])
	if msg.code == syncRequest {
		msg.payload = message.GetChunk{
			Start: binary.BigEndian.Uint64(buf[1:9]),
			End:   binary.BigEndian.Uint64(buf[9:17]),
		}
	}
	return
}

func (f *syncConn) write(msg syncMsg) error {
	buf := make([]byte, 20)
	buf[0] = byte(msg.code)

	if msg.code == syncRequest {
		chunk := msg.payload.(message.GetChunk)
		binary.BigEndian.PutUint64(buf[1:9], chunk.Start)
		binary.BigEndian.PutUint64(buf[9:17], chunk.End)
	}

	_, err := f.Conn.Write(buf)
	return err
}

func newFileConn(conn net2.Conn, id vnode.NodeID, cacher syncCacher, log log15.Logger) *syncConn {
	return &syncConn{
		Conn:   conn,
		id:     id,
		cacher: cacher,
		buf:    make([]byte, 1024),
		log:    log,
	}
}

func (f *syncConn) status() FileConnStatus {
	return FileConnStatus{
		Id:    f.id.String(),
		Addr:  f.RemoteAddr().String(),
		Speed: f.speed,
	}
}

func (f *syncConn) isBusy() bool {
	return atomic.LoadInt32(&f.busy) == 1
}

func (f *syncConn) download(from, to uint64) (err error) {
	f.setBusy()
	defer f.idle()

	f.log.Info(fmt.Sprintf("begin download chunk %d-%d from %s", from, to, f.RemoteAddr()))

	err = f.write(syncMsg{
		code: syncRequest,
		payload: message.GetChunk{
			Start: from,
			End:   to,
		},
	})

	if err != nil {
		f.log.Error(fmt.Sprintf("pack get chunk %d-%d to %s error: %v", from, to, f.RemoteAddr(), err))
		return &downloadError{
			code: downloadPackMsgErr,
			err:  err.Error(),
		}
	}

	var msg syncMsg
	msg, err = f.read()
	if err != nil {
		return
	}

	if msg.code != syncReady {
		return errors.New("remote not ready")
	}

	cache := f.cacher.GetSyncCache()
	writer, err := cache.NewWriter(from, to)
	if err != nil {
		return err
	}

	defer writer.Close()

	start := time.Now()
	var nr, nw, total int
	var werr error
	for {
		//_ = f.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		nr, err = f.Conn.Read(f.buf)
		total += nr

		nw, werr = writer.Write(f.buf[:nr])
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		} else if werr != nil || nw != nr {
			return errors.New("write too short")
		}
	}

	f.speed = float64(total) / (time.Now().Sub(start).Seconds() + 1)

	return nil
}

func (f *syncConn) setBusy() {
	atomic.StoreInt32(&f.busy, 1)
	atomic.StoreInt64(&f.t, time.Now().Unix())
}

func (f *syncConn) idle() {
	atomic.StoreInt32(&f.busy, 0)
	atomic.StoreInt64(&f.t, time.Now().Unix())
}

func (f *syncConn) close() error {
	if atomic.CompareAndSwapInt32(&f.closed, 0, 1) {
		return f.Conn.Close()
	}

	return errFileConnClosed
}
