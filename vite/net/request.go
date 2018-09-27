package net

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
	"time"
)

type Request interface {
	handle(cmd cmd, data p2p.Serializable, peer *Peer)
	data() (sblocks []*ledger.SnapshotBlock, ablocks map[types.Address][]*ledger.AccountBlock)
	childDone(r Request)
	id() uint64
	pid() uint64
	setId(id uint64)
	expired() bool
	run(peers *peerSet, isRetry bool) bool
	peer() *Peer
}

type RequestPool interface {
	Add(r Request)
	Retry(r Request)
	Done(r Request)
}

type subLedgerRequest struct {
	_id, _pid uint64	// unique id
	start, end uint64	// snapshotblock startHeight and endHeight
	count uint64	// current amount of snapshotblocks have got
	children                     []Request
	sblocks                      []*ledger.SnapshotBlock
	mblocks                      map[types.Address][]*ledger.AccountBlock
	pool                         RequestPool
	_peer                         *Peer
	expiration                   time.Time
}

func (s *subLedgerRequest) peer() *Peer {
	return s._peer
}

func (s *subLedgerRequest) run(peers *peerSet, isRetry bool) bool {
	if isRetry {
		peer := peers.BestPeer()
		s._peer = peer
	} else {
		peers := peers.Pick(s.end)
		if len(peers) > 0 {
			s._peer = peers[0]
		}
	}

	if s._peer != nil {
		err := s._peer.GetSubLedger(&message.GetSnapshotBlocks{
			From:    &ledger.HashHeight{
				Height: s.start,
			},
			Count: s.end - s.start + 1,
			Forward: true,
		})

		if err == nil {
			return true
		}
	}

	return false
}

func (s *subLedgerRequest) setId(id uint64) {
	s._id = id
}

func (s *subLedgerRequest) id() uint64 {
	return s._id
}

func (s *subLedgerRequest) pid() uint64 {
	return s._pid
}

func (s *subLedgerRequest) expired() bool {
	return time.Now().After(s.expiration)
}

func (s *subLedgerRequest) childDone(r Request) {
	sblocks, mblocks := r.data()

	for _, sblock := range sblocks {
		offset := sblock.Height - s.start
		if s.sblocks[offset] == nil {
			s.sblocks[offset] = sblock
			s.count++
		}
	}

	if s.mblocks == nil {
		s.mblocks = make(map[types.Address][]*ledger.AccountBlock)
	}

	for address, ablocks := range mblocks {
		s.mblocks[address] = append(s.mblocks[address], ablocks...)
	}

	for i, cr := range s.children {
		if cr == r {
			s.children[i] = s.children[len(s.children)-1]
			s.children = s.children[:len(s.children)-1]
		}
	}

	// all children have done & got all snapshotblocks in need
	if len(s.children) == 0 && s.count == (s.end - s.start) {
		s.pool.Done(s)
	}
}

func (s *subLedgerRequest) handle(cmd cmd, data p2p.Serializable, peer *Peer) {
	switch cmd {
	case SubLedgerCode:
		msg, _ := data.(*message.SubLedger)
		if uint64(len(msg.SBlocks)) == (s.end - s.start + 1) {
			s.sblocks = msg.SBlocks
			s.mblocks = msg.ABlocks
			s.pool.Done(s)
		} else {
			s.pool.Retry(s)
		}
	case FileListCode:
		msg, _ := data.(*message.FileList)
		for _, file := range msg.Files {
			r := &fileRequest{
				_pid:       s._id,
				nonce:      msg.Nonce,
				file:       file,
				expiration: time.Now().Add(2 * time.Minute),
			}
			s.pool.Add(r)
		}

		for _, chunk := range msg.Chunk {
			if chunk[1] - chunk[0] != 0 {
				c := &chunkRequest{
					_pid:       s._id,
					start:      chunk[0],
					end:        chunk[1],
					_peer:       peer,
					expiration: time.Now().Add(30 * time.Second),
				}
				s.pool.Add(c)
			}
		}
	}
}

func (s *subLedgerRequest) data() (sblocks []*ledger.SnapshotBlock, mblocks map[types.Address][]*ledger.AccountBlock) {
	return s.sblocks, s.mblocks
}

// @request for file
type fileRequest struct {
	_id, _pid, count, nonce uint64
	file *message.File
	sblocks []*ledger.SnapshotBlock
	mblocks map[types.Address][]*ledger.AccountBlock
	_peer *Peer
	expiration time.Time
}

func (f *fileRequest) handle(cmd cmd, data p2p.Serializable, peer *Peer) {
	panic("implement me")
}

func (f *fileRequest) data() (sblocks []*ledger.SnapshotBlock, ablocks map[types.Address][]*ledger.AccountBlock) {
	panic("implement me")
}

func (f *fileRequest) childDone(r Request) {
	panic("implement me")
}

func (f *fileRequest) id() uint64 {
	panic("implement me")
}

func (f *fileRequest) pid() uint64 {
	panic("implement me")
}

func (f *fileRequest) setId(id uint64) {
	panic("implement me")
}

func (f *fileRequest) expired() bool {
	panic("implement me")
}

func (f *fileRequest) run(peers *peerSet, isRetry bool) bool {
	panic("implement me")
}

func (f *fileRequest) peer() *Peer {
	panic("implement me")
}

// @request for chunk
type chunkRequest struct {
	_id, _pid, count uint64
	start, end uint64
	sblocks []*ledger.SnapshotBlock
	mblocks map[types.Address][]*ledger.AccountBlock
	_peer *Peer
	expiration time.Time
}

func (c *chunkRequest) handle(cmd cmd, data p2p.Serializable, peer *Peer) {
	panic("implement me")
}

func (c *chunkRequest) data() (sblocks []*ledger.SnapshotBlock, ablocks map[types.Address][]*ledger.AccountBlock) {
	panic("implement me")
}

func (c *chunkRequest) childDone(r Request) {
	panic("implement me")
}

func (c *chunkRequest) id() uint64 {
	panic("implement me")
}

func (c *chunkRequest) pid() uint64 {
	panic("implement me")
}

func (c *chunkRequest) setId(id uint64) {
	panic("implement me")
}

func (c *chunkRequest) expired() bool {
	panic("implement me")
}

func (c *chunkRequest) run(peers *peerSet, isRetry bool) bool {
	panic("implement me")
}

func (c *chunkRequest) peer() *Peer {
	panic("implement me")
}
