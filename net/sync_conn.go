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
	"bytes"
	"encoding/binary"
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
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/net/vnode"
	"github.com/vitelabs/go-vite/vitepb"
)

var errWriteTooShort = errors.New("write too short")
var errHandshakeError = errors.New("sync handshake error")
var errServerNotReady = errors.New("server not ready")
var errIncompleteChunk = errors.New("incomplete chunk")

type syncHandshake struct {
	id    peerId
	key   []byte
	time  int64
	token []byte
}

func (s *syncHandshake) Serialize() ([]byte, error) {
	pb := &vitepb.SyncConnHandshake{
		ID:        s.id.Bytes(),
		Timestamp: s.time,
		Key:       s.key,
		Token:     s.token,
	}
	return proto.Marshal(pb)
}

func (s *syncHandshake) deserialize(data []byte) error {
	pb := &vitepb.SyncConnHandshake{}
	err := proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}

	s.id, err = vnode.Bytes2NodeID(pb.ID)
	if err != nil {
		return err
	}
	s.key = pb.Key
	s.time = pb.Timestamp
	s.token = pb.Token
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
	initiate(conn net2.Conn, peer *Peer) (*syncConn, error)
}

type defaultSyncConnectionFactory struct {
	chain   syncCacher
	peers   *peerSet
	id      peerId
	peerKey ed25519.PrivateKey
	mineKey ed25519.PrivateKey
}

func (d *defaultSyncConnectionFactory) makeSyncConn(conn net2.Conn) *syncConn {
	return &syncConn{
		conn: conn,
		c:    NewTransport(conn, 100, 10*time.Second, 10*time.Second),
	}
}

func (d *defaultSyncConnectionFactory) initiate(conn net2.Conn, peer *Peer) (*syncConn, error) {
	c := d.makeSyncConn(conn)

	hk := &syncHandshake{
		id:   d.id,
		time: time.Now().Unix(),
	}
	pub := ed25519.PublicKey(peer.Id.Bytes()).ToX25519Pk()
	priv := d.peerKey.ToX25519Sk()
	secret, err := crypto.X25519ComputeSecret(priv, pub)
	if err != nil {
		return nil, err
	}

	t := make([]byte, 8)
	binary.BigEndian.PutUint64(t, uint64(hk.time))
	hash := crypto.Hash256(t)
	hk.token = xor(hash, secret)
	if len(d.mineKey) != 0 {
		hk.key = d.mineKey.PubByte()
		hk.token = ed25519.Sign(d.mineKey, hk.token)
	}

	data, err := hk.Serialize()
	if err != nil {
		return nil, err
	}

	err = c.c.WriteMsg(Msg{
		Code:    CodeSyncHandshake,
		Payload: data,
	})

	if err != nil {
		return nil, err
	}

	msg, err := c.c.ReadMsg()
	if err != nil {
		return nil, err
	}

	if msg.Code != CodeSyncHandshakeOK {
		return nil, errHandshakeError
	}

	c.peer = peer
	c.cacher = d.chain

	return c, nil
}

func (d *defaultSyncConnectionFactory) receive(conn net2.Conn) (*syncConn, error) {
	c := d.makeSyncConn(conn)

	msg, err := c.c.ReadMsg()
	if err != nil {
		return nil, err
	}

	if msg.Code != CodeSyncHandshake {
		_ = c.c.WriteMsg(Msg{
			Code:    CodeDisconnect,
			Payload: []byte{byte(PeerNotHandshakeMsg)},
		})
		return nil, PeerNotHandshakeMsg
	}

	var hk = &syncHandshake{}
	err = hk.deserialize(msg.Payload)
	if err != nil {
		return nil, err
	}

	pub := ed25519.PublicKey(hk.id.Bytes()).ToX25519Pk()
	priv := d.peerKey.ToX25519Sk()
	secret, err := crypto.X25519ComputeSecret(priv, pub)
	if err != nil {
		return nil, err
	}

	t := make([]byte, 8)
	binary.BigEndian.PutUint64(t, uint64(hk.time))
	hash := crypto.Hash256(t)
	token := xor(hash, secret)
	if len(hk.key) != 0 {
		if false == ed25519.Verify(hk.key, token, hk.token) {
			_ = c.c.WriteMsg(Msg{
				Code:    CodeDisconnect,
				Payload: []byte{byte(PeerInvalidSignature)},
			})
			return nil, PeerInvalidSignature
		}
	} else if false == bytes.Equal(hk.token, token) {
		_ = c.c.WriteMsg(Msg{
			Code:    CodeDisconnect,
			Payload: []byte{byte(PeerInvalidToken)},
		})
		return nil, PeerInvalidToken
	}

	p := d.peers.get(hk.id)
	if p == nil {
		_ = c.c.WriteMsg(Msg{
			Code:    CodeDisconnect,
			Payload: []byte{byte(PeerNoPermission)},
		})
		return nil, PeerNoPermission
	}

	err = c.c.WriteMsg(Msg{
		Code: CodeSyncHandshakeOK,
	})
	if err != nil {
		return nil, err
	}

	c.peer = p
	c.cacher = d.chain

	return c, nil
}

type syncConn struct {
	conn   net2.Conn
	c      Codec
	peer   *Peer
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
		Address: f.peer.Id.Brief() + "@" + f.address(),
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

func isRightChunk(msg *syncResponse, t *syncTask) (seg interfaces.Segment, err error) {
	if msg.from != t.From || msg.to != t.To {
		err = fmt.Errorf("bound not equal: %d-%d %d-%d", msg.from, msg.to, t.From, t.To)
		return
	}

	if msg.prevHash != t.PrevHash || msg.endHash != t.Hash {
		err = fmt.Errorf("hash not equal: %s-%s %s-%s", msg.prevHash, msg.endHash, t.PrevHash, t.Hash)
		return
	}

	return t.Segment, err
}

func (f *syncConn) download(t *syncTask) (fatal bool, err error) {
	if false == atomic.CompareAndSwapInt32(&f.busy, 0, 1) {
		err = fmt.Errorf("task %s is downloading", f.task.String())
		return
	}
	defer atomic.StoreInt32(&f.busy, 0)
	f.task = *t

	request := &syncRequest{
		from:     t.From,
		to:       t.To,
		prevHash: t.PrevHash,
		endHash:  t.Hash,
	}
	data, err := request.Serialize()
	if err != nil {
		return false, err
	}

	err = f.c.WriteMsg(Msg{
		Code:    CodeSyncRequest,
		Payload: data,
	})

	if err != nil {
		return true, err
	}

	msg, err := f.c.ReadMsg()
	if err != nil {
		return true, err
	}

	if msg.Code != CodeSyncReady {
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
	writer, err := cache.NewWriter(segment, int64(chunkInfo.size))
	if err != nil {
		return false, err
	}

	start := time.Now().Unix()
	var nr, nw int
	var total, count uint64
	var rerr, werr error

	if f._speed < 10240 { // 10k/s
		f._speed = 10240
	}

	timeout := time.Duration(2*chunkInfo.size/f._speed) * time.Second
	_ = f.conn.SetReadDeadline(time.Now().Add(timeout))
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

	err = writer.Close()
	if err != nil {
		fatal = false
		_ = cache.Delete(segment)
		return
	}

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

	t.source = f.peer.Id
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

type downloadConnPool struct {
	mu        sync.Mutex
	peers     *peerSet
	mi        map[peerId]int // value is the index of `connPoolImpl.l`
	l         connections    // connections sort by speed, from fast to slow
	blackList map[peerId]int64
}

func newDownloadConnPool(peers *peerSet) *downloadConnPool {
	return &downloadConnPool{
		peers:     peers,
		mi:        make(map[peerId]int),
		blackList: make(map[peerId]int64),
	}
}

func (fp *downloadConnPool) blockPeer(id peerId, duration time.Duration) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.blackList[id] = time.Now().Add(duration).Unix()
}

func (fp *downloadConnPool) blocked(id peerId) bool {
	now := time.Now().Unix()

	fp.mu.Lock()
	defer fp.mu.Unlock()

	t, ok := fp.blackList[id]
	if !ok {
		return false
	}

	if t > now {
		return true
	}

	delete(fp.blackList, id)
	return false
}

func (fp *downloadConnPool) connections() []SyncConnectionStatus {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	cs := make([]SyncConnectionStatus, len(fp.l))

	for i := 0; i < len(fp.l); i++ {
		cs[i] = fp.l[i].status()
	}

	return cs
}

// delete filePeer and connection
func (fp *downloadConnPool) delConn(c *syncConn) {
	_ = c.close()

	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.delConnLocked(c.peer.Id)
}

func (fp *downloadConnPool) delConnLocked(id peerId) {
	if i, ok := fp.mi[id]; ok {
		delete(fp.mi, id)

		fp.l = fp.l.del(i)
	}
}

func (fp *downloadConnPool) addConn(c *syncConn) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if _, ok := fp.mi[c.peer.Id]; ok {
		return errSyncConnExist
	}

	fp.l = append(fp.l, c)
	fp.mi[c.peer.Id] = len(fp.l) - 1
	return nil
}

// sort list, and update index to map
func (fp *downloadConnPool) sort() {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.sortLocked()
}

func (fp *downloadConnPool) sortLocked() {
	sort.Sort(fp.l)
	for i, c := range fp.l {
		fp.mi[c.peer.Id] = i
	}
}

// choose the fast fileConn, or create new conn randomly
func (fp *downloadConnPool) chooseSource(t *syncTask) (*Peer, *syncConn, error) {
	peerMap := fp.peers.pickDownloadPeers(t.To)

	if len(peerMap) == 0 {
		return nil, nil, errNoSuitablePeer
	}

	fp.mu.Lock()
	defer fp.mu.Unlock()

	// only peers without sync connection
	for _, c := range fp.l {
		delete(peerMap, c.peer.Id)
	}

	// is in blackList
	now := time.Now().Unix()
	for k, p := range peerMap {
		if tt, ok := fp.blackList[p.Id]; ok && now > tt {
			delete(peerMap, k)
		}
	}

	var createNew bool
	if len(peerMap) > 0 {
		createNew = rand.Intn(10) > 5
	}

	fp.sortLocked()
	for i, c := range fp.l {
		if c.isBusy() || c.peer.Height < t.To {
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

func (fp *downloadConnPool) reset() {
	fp.mu.Lock()

	fp.mi = make(map[peerId]int)
	l := fp.l
	fp.l = nil

	fp.mu.Unlock()

	for _, c := range l {
		_ = c.close()
	}
}

func xor(one, other []byte) (xor []byte) {
	xor = make([]byte, len(one))
	for i := 0; i < len(one); i++ {
		xor[i] = one[i] ^ other[i]
	}
	return xor
}
