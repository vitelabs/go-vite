/*
 * Copyright 2019 The go-vite Authors
 * This file is part of the go-vite library.
 *
 * The go-vite library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The go-vite library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with the go-vite library. If not, see <http://www.gnu.org/licenses/>.
 */

package net

import (
	"errors"
	"fmt"
	"math/rand"
	net2 "net"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/p2p/vnode"
	"github.com/vitelabs/go-vite/vitepb"
)

var errWriteTooShort = errors.New("write too short")
var errHandshakeError = errors.New("sync handshake error")
var errServerNotReady = errors.New("server not ready")
var errIncompleteChunk = errors.New("incomplete chunk")

type syncHandshake struct {
	key  []byte
	time time.Time
	sign []byte
}

func (s *syncHandshake) Serialize() ([]byte, error) {
	pb := &vitepb.SyncConnHandshake{
		Key:       s.key,
		Timestamp: s.time.Unix(),
		Sign:      s.sign,
	}
	return proto.Marshal(pb)
}

func (s *syncHandshake) deserialize(data []byte) error {
	pb := &vitepb.SyncConnHandshake{}
	err := proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}

	s.key = pb.Key
	s.time = time.Unix(pb.Timestamp, 0)
	s.sign = pb.Sign
	return nil
}

type syncRequest struct {
	from, to          uint64
	prevHash, endHash types.Hash
}

func (s *syncRequest) Serialize() ([]byte, error) {
	pb := &vitepb.ChunkRequest{
		From:     s.from,
		To:       s.to,
		PrevHash: s.prevHash.Bytes(),
		EndHash:  s.endHash.Bytes(),
	}

	return proto.Marshal(pb)
}

func (s *syncRequest) deserialize(data []byte) error {
	pb := &vitepb.ChunkRequest{}
	err := proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}
	s.from = pb.From
	s.to = pb.To

	s.prevHash, err = types.BytesToHash(pb.PrevHash)
	if err != nil {
		return err
	}

	s.endHash, err = types.BytesToHash(pb.EndHash)

	return err
}

type syncResponse struct {
	from, to          uint64
	size              uint64
	prevHash, endHash types.Hash
}

func (s *syncResponse) Serialize() ([]byte, error) {
	pb := &vitepb.ChunkResponse{
		From:     s.from,
		To:       s.to,
		Size:     s.size,
		PrevHash: s.prevHash.Bytes(),
		EndHash:  s.endHash.Bytes(),
	}

	return proto.Marshal(pb)
}

func (s *syncResponse) deserialize(data []byte) error {
	pb := &vitepb.ChunkResponse{}
	err := proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}

	s.prevHash, err = types.BytesToHash(pb.PrevHash)
	if err != nil {
		return err
	}

	s.endHash, err = types.BytesToHash(pb.EndHash)
	if err != nil {
		return err
	}

	s.from = pb.From
	s.to = pb.To
	s.size = pb.Size
	return nil
}

type SyncConnectionStatus struct {
	Address string `json:"address"`
	Speed   string `json:"speed"`
	Task    string `json:"task"`
}

type syncConnReceiver interface {
	receive(conn net2.Conn) (*syncConn, error)
}

type syncConnInitiator interface {
	initiate(conn net2.Conn, peer Peer) (*syncConn, error)
}

type defaultSyncConnectionFactory struct {
	chain      syncCacher
	peers      *peerSet
	privateKey ed25519.PrivateKey
}

func (d *defaultSyncConnectionFactory) makeCodec(conn net2.Conn) *syncConn {
	return &syncConn{
		conn: conn,
		c:    p2p.NewTransport(conn, 100, 10*time.Second, 10*time.Second),
	}
}

func (d *defaultSyncConnectionFactory) initiate(conn net2.Conn, peer Peer) (*syncConn, error) {
	c := d.makeCodec(conn)

	hk := &syncHandshake{
		key:  d.privateKey.PubByte(),
		time: time.Now(),
		sign: nil,
	}

	data, err := hk.Serialize()
	if err != nil {
		return nil, err
	}

	err = c.c.WriteMsg(p2p.Msg{
		Code:    p2p.CodeSyncHandshake,
		Payload: data,
	})

	if err != nil {
		return nil, err
	}

	msg, err := c.c.ReadMsg()
	if err != nil {
		return nil, err
	}

	if msg.Code != p2p.CodeSyncHandshakeOK {
		return nil, errHandshakeError
	}

	c.Peer = peer
	c.cacher = d.chain

	return c, nil
}

func (d *defaultSyncConnectionFactory) receive(conn net2.Conn) (*syncConn, error) {
	c := d.makeCodec(conn)

	msg, err := c.c.ReadMsg()
	if err != nil {
		return nil, err
	}

	if msg.Code != p2p.CodeSyncHandshake {
		_ = c.c.WriteMsg(p2p.Msg{
			Code:    p2p.CodeDisconnect,
			Payload: []byte{byte(p2p.PeerNotHandshakeMsg)},
		})
		return nil, errHandshakeError
	}

	var hk = &syncHandshake{}
	err = hk.deserialize(msg.Payload)
	if err != nil {
		return nil, err
	}

	id, err := vnode.Bytes2NodeID(hk.key)
	if err != nil {
		_ = c.c.WriteMsg(p2p.Msg{
			Code:    p2p.CodeDisconnect,
			Payload: []byte{byte(p2p.PeerInvalidMessage)},
		})
		return nil, errHandshakeError
	}

	p := d.peers.get(id)
	if p == nil {
		_ = c.c.WriteMsg(p2p.Msg{
			Code:    p2p.CodeDisconnect,
			Payload: []byte{byte(p2p.PeerNoPermission)},
		})
		return nil, errHandshakeError
	}

	err = c.c.WriteMsg(p2p.Msg{
		Code: p2p.CodeSyncHandshakeOK,
	})
	if err != nil {
		return nil, err
	}

	c.Peer = p
	c.cacher = d.chain

	return c, nil
}

type syncConn struct {
	conn net2.Conn
	c    p2p.Codec
	Peer
	busy   int32  // atomic
	_speed uint64 // download speed, byte/s
	task   syncTask
	closed int32
	cacher syncCacher
	buf    [1024]byte
	failed int32
}

var speedUnits = [...]string{
	" Byte/s",
	" KByte/s",
	" MByte/s",
	" GByte/s",
}

func formatSpeed(s float64) (sf float64, unit int) {
	for unit = 1; unit < len(speedUnits); unit++ {
		if sf = s / 1024.0; sf > 1 {
			s = sf
		} else {
			break
		}
	}

	unit--

	return s, unit
}

func speedToString(s float64) string {
	sf, unit := formatSpeed(s)

	return strconv.FormatFloat(sf, 'f', 2, 64) + speedUnits[unit]
}

func (f *syncConn) status() SyncConnectionStatus {
	st := SyncConnectionStatus{
		Address: f.ID().Brief() + "@" + f.address(),
		Speed:   speedToString(float64(f._speed)),
		Task:    "",
	}

	if f.isBusy() {
		st.Task = f.task.String()
	}

	return st
}

func (f *syncConn) address() string {
	return f.conn.RemoteAddr().String()
}

func (f *syncConn) fail() bool {
	f.failed++

	return f.failed > 3
}

func (f *syncConn) speed() uint64 {
	return f._speed
}

func (f *syncConn) isBusy() bool {
	return atomic.LoadInt32(&f.busy) == 1
}

func isRightChunk(msg *syncResponse, t syncTask) (seg interfaces.Segment, err error) {
	if msg.from != t.from || msg.to != t.to {
		err = fmt.Errorf("different bound: %d-%d %d-%d", msg.from, msg.to, t.from, t.to)
		return
	}

	if msg.prevHash != t.prevHash || msg.endHash != t.endHash {
		err = fmt.Errorf("hash not equal: %s-%s %s-%s", msg.prevHash, msg.endHash, t.prevHash, t.endHash)
		return
	}

	seg.Bound = [2]uint64{t.from, t.to}
	seg.PrevHash = t.prevHash
	seg.Hash = t.endHash

	return
}

func (f *syncConn) download(t syncTask) (fatal bool, err error) {
	if false == atomic.CompareAndSwapInt32(&f.busy, 0, 1) {
		err = fmt.Errorf("task %s is downloading", f.task.String())
		return
	}
	defer atomic.StoreInt32(&f.busy, 0)
	f.task = t

	request := &syncRequest{
		from:     t.from,
		to:       t.to,
		prevHash: t.prevHash,
		endHash:  t.endHash,
	}
	data, err := request.Serialize()
	if err != nil {
		return false, err
	}

	err = f.c.WriteMsg(p2p.Msg{
		Code:    p2p.CodeSyncRequest,
		Payload: data,
	})

	if err != nil {
		return true, err
	}

	msg, err := f.c.ReadMsg()
	if err != nil {
		return true, err
	}

	if msg.Code != p2p.CodeSyncReady {
		fatal = f.fail()
		return fatal, errServerNotReady
	}

	var chunkInfo = &syncResponse{}
	err = chunkInfo.deserialize(msg.Payload)
	if err != nil {
		return true, err
	}

	segment, err := isRightChunk(chunkInfo, t)
	if err != nil {
		return true, err
	}

	cache := f.cacher.GetSyncCache()
	writer, err := cache.NewWriter(segment)
	if err != nil {
		return false, err
	}

	start := time.Now().Unix()
	var nr, nw int
	var total, count uint64
	var rerr, werr error
	_ = f.conn.SetReadDeadline(time.Now().Add(fileTimeout))
	for {
		count = chunkInfo.size - total
		if count > 1024 {
			count = 1024
		}

		nr, rerr = f.conn.Read(f.buf[:count])
		total += uint64(nr)

		nw, werr = writer.Write(f.buf[:nr])

		if rerr != nil {
			break
		} else if werr != nil {
			break
		} else if nw != nr {
			werr = errWriteTooShort
			break
		}

		if total == chunkInfo.size {
			break
		}
	}

	_ = writer.Close()

	if rerr != nil {
		fatal = true
	}

	if werr != nil {
		err = fmt.Errorf("failed to write cache %s: %v", t.String(), werr)
		_ = cache.Delete(segment)
		return
	}

	if total != chunkInfo.size {
		fatal = true
		err = errIncompleteChunk
		_ = cache.Delete(segment)
		return
	}

	f._speed = total / uint64(time.Now().Unix()-start+1)

	return
}

func (f *syncConn) close() error {
	if atomic.CompareAndSwapInt32(&f.closed, 0, 1) {
		return f.conn.Close()
	}

	return errSyncConnClosed
}

var errSyncConnExist = errors.New("sync connection has exist")
var errSyncConnClosed = errors.New("sync connection has closed")
var errPeerDialing = errors.New("peer is dialing")

type connections []*syncConn

func (fl connections) Len() int {
	return len(fl)
}

func (fl connections) Less(i, j int) bool {
	return fl[i].speed() > fl[j].speed()
}

func (fl connections) Swap(i, j int) {
	fl[i], fl[j] = fl[j], fl[i]
}

func (fl connections) del(i int) connections {
	total := len(fl)
	if i < total {
		copy(fl[i:], fl[i+1:])
		return fl[:total-1]
	}

	return fl
}

type FilePoolStatus struct {
	Connections []SyncConnectionStatus `json:"connections"`
}

type connPoolImpl struct {
	mu    sync.Mutex
	peers *peerSet
	mi    map[peerId]int // value is the index of `connPoolImpl.l`
	l     connections    // connections sort by speed, from fast to slow
}

func newPool(peers *peerSet) *connPoolImpl {
	return &connPoolImpl{
		mi:    make(map[peerId]int),
		peers: peers,
	}
}

func (fp *connPoolImpl) connections() []SyncConnectionStatus {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	cs := make([]SyncConnectionStatus, len(fp.l))

	for i := 0; i < len(fp.l); i++ {
		cs[i] = fp.l[i].status()
	}

	return cs
}

// delete filePeer and connection
func (fp *connPoolImpl) delConn(c *syncConn) {
	_ = c.close()

	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.delConnLocked(c.ID())
}

func (fp *connPoolImpl) delConnLocked(id peerId) {
	if i, ok := fp.mi[id]; ok {
		delete(fp.mi, id)

		fp.l = fp.l.del(i)
	}
}

func (fp *connPoolImpl) addConn(c *syncConn) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if _, ok := fp.mi[c.ID()]; ok {
		return errSyncConnExist
	}

	fp.l = append(fp.l, c)
	fp.mi[c.ID()] = len(fp.l) - 1
	return nil
}

// sort list, and update index to map
func (fp *connPoolImpl) sort() {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.sortLocked()
}

func (fp *connPoolImpl) sortLocked() {
	sort.Sort(fp.l)
	for i, c := range fp.l {
		fp.mi[c.ID()] = i
	}
}

// choose the fast fileConn, or create new conn randomly
func (fp *connPoolImpl) chooseSource(t *syncTask) (Peer, *syncConn, error) {
	peerMap := fp.peers.pickDownloadPeers(t.to)

	if len(peerMap) == 0 {
		return nil, nil, errNoSuitablePeer
	}

	fp.mu.Lock()
	defer fp.mu.Unlock()

	// only peers without sync connection
	for _, c := range fp.l {
		delete(peerMap, c.ID())
	}

	var createNew bool
	if len(peerMap) > 0 {
		createNew = rand.Intn(10) > 5
	}

	fp.sortLocked()
	for i, c := range fp.l {
		if c.isBusy() || c.Height() < t.to {
			continue
		}

		if len(fp.l)+1 > 3*(i+1) {
			// fast enough
			return nil, c, nil
		}

		if createNew {
			for _, p := range peerMap {
				return p, nil, nil
			}
		} else {
			return nil, c, nil
		}
	}

	for _, p := range peerMap {
		return p, nil, nil
	}

	return nil, nil, nil
}

func (fp *connPoolImpl) reset() {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.mi = make(map[peerId]int)

	for _, c := range fp.l {
		_ = c.close()
	}

	fp.l = nil
}
