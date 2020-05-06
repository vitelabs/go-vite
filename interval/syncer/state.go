package syncer

import (
	"sync"

	"time"

	"encoding/json"

	"github.com/asaskevich/EventBus"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/p2p"
)

type syncPeer struct {
	peer p2p.Peer
	msg  *stateMsg
}
type syncTask struct {
	height uint64
	hash   string
	peer   p2p.Peer
	closed chan struct{}
	done   int
}

type state struct {
	rw      *chainRw
	peers   sync.Map
	fetcher *fetcher
	sender  *sender
	closed  chan struct{}
	wg      sync.WaitGroup
	syncNum int
	firstTa *syncTask
	p       p2p.P2P
	bus     EventBus.Bus
}

type handState struct {
	GenesisHash string
	S           peerState
}

func (self *state) GetState() (interface{}, error) {
	return self.getHandState()
}

func (self *state) Handshake(peerId string, state []byte) error {
	msg := &handState{}
	err := json.Unmarshal(state, msg)
	if err != nil {
		return err
	}
	hs, err := self.getHandState()
	if err != nil {
		return err
	}
	if hs.GenesisHash != msg.GenesisHash {
		return errors.New("Genesis diff, self[" + hs.GenesisHash + "], peer[" + msg.GenesisHash + "]")
	}
	if msg.S.Hash == "" {
		return errors.New("Snapshot Hash empty.")
	}
	log.Info("handshake success, peer[%s] snapshot height:%d", peerId, msg.S.Height)
	return nil
}

func (self *state) getHandState() (*handState, error) {
	head, e := self.rw.HeadSnapshot()
	if e != nil {
		return nil, errors.New("get snapshot head block fail.")

	}
	genesis, e := self.rw.GenesisSnapshot()
	if e != nil {
		return nil, errors.New("get genesis block fail.")
	}
	msg := &handState{GenesisHash: genesis.Hash(), S: peerState{Height: head.Height(), Hash: head.Hash()}}
	return msg, nil
}

func (self *state) DecodeState(state []byte) interface{} {
	if state == nil {
		return nil
	}
	msg := &handState{}
	e := json.Unmarshal(state, msg)
	if e != nil {
		log.Error("DecodeState fail. err:%v", e)
		return nil
	}
	return msg
}

func (self *state) EncodeState(state interface{}) []byte {
	if state == nil {
		return nil
	}
	b, e := json.Marshal(state)
	if e != nil {
		log.Error("EncodeState fail. err:%v", e)
		return nil
	}
	return b
}

func newState(rw *chainRw, fetcher *fetcher, s *sender, p p2p.P2P, bus EventBus.Bus) *state {
	self := &state{}
	self.rw = rw
	self.fetcher = fetcher
	self.sender = s
	self.closed = make(chan struct{})
	self.p = p
	self.firstTa = &syncTask{closed: make(chan struct{})}
	self.bus = bus
	return self
}

func (self *state) update(msg *stateMsg, peer p2p.Peer) {
	//syncP := self.peers[peer.Id()]
	//if syncP == nil {
	//	log.Warn("peer[%s] not exist", peer.Id())
	//	return
	//}
	//m := syncP.msg
	//if m == nil {
	//	syncP.msg = m
	//}
	prevState := peer.GetState()
	if prevState == nil {
		log.Error("peer state is empty.", peer.Id())
	} else {
		state := prevState.(*handState)
		state.S.Height = msg.Height
		state.S.Hash = msg.Hash
	}
	head, e := self.rw.HeadSnapshot()
	if e != nil {
		log.Error("read snapshot head error:%v", e)
		return
	}
	if msg.Height > head.Height() {
		self.fetcher.fetchSnapshotBlockFromPeer(common.HashHeight{Hash: msg.Hash, Height: msg.Height}, peer)
	}
}
func (self *state) peerConnected(peer p2p.Peer) {
	self.peers.Store(peer.Id(), &syncPeer{peer: peer})
}
func (self *state) peerClosed(peer p2p.Peer) {
	self.peers.Delete(peer.Id())
}
func (self *state) start() {
	go self.loop()
	go self.syncFirst()
}

func (self *state) loop() {
	self.wg.Add(1)
	defer self.wg.Done()
	ticker := time.NewTicker(time.Second * 20)
	for {
		select {
		case <-self.closed:
			return
		case <-ticker.C:
			head, e := self.rw.HeadSnapshot()
			if e != nil {
				log.Error("read snapshot head error:%v", e)
				continue
			}
			stateMsg := stateMsg{Hash: head.Hash(), Height: head.Height()}
			log.Info("sync state, state:%v", stateMsg)
			self.sender.broadcastState(stateMsg)
		}
	}
}

func (self *state) syncDone() bool {
	return self.firstTa.done > 0
}

func (self *state) stop() {
	close(self.closed)
	self.wg.Wait()
}
func (self *state) syncFirst() {
	self.wg.Add(1)
	defer self.wg.Done()
	ta := self.firstTa

	t := time.NewTicker(time.Second * 2)

	timeout := time.NewTicker(time.Second * 50)
NET:
	for {
		select {
		case <-self.closed:
			return
		case <-ta.closed:
			return
		case <-timeout.C:
			log.Info("link other node fail. but started.")
			self.firstSyncDone()
		case <-t.C:
			i := 0
			self.peers.Range(func(_, _ interface{}) bool {
				i++
				return true
			})
			if i > 0 {
				p := self.bestPeer()
				if p != nil {
					s := p.GetState().(*handState)
					ta.hash = s.S.Hash
					ta.height = s.S.Height
					ta.peer = p
					break NET
				}

			}
		}
	}

	head, e := self.rw.HeadSnapshot()
	if e != nil {
		log.Error("read snapshot head error:%v", e)
		return
	}
	if head.Height() >= ta.height {
		log.Info("need not sync, self height:%d, peer height:%d, peer id:%s", head.Height(), ta.height, ta.peer.Id())
		self.firstSyncDone()
		return
	}
	self.sender.requestSnapshotBlockByPeer(common.HashHeight{Hash: ta.hash, Height: ta.height}, ta.peer)

	for {
		select {
		case <-self.closed:
			return
		case <-ta.closed:
			return
		case <-t.C:
			head, e := self.rw.HeadSnapshot()
			if e != nil {
				log.Error("read snapshot head error:%v", e)
				continue
			}
			if head.Height() > ta.height {
				log.Info("sync first finish.")
				self.firstSyncDone()
				return
			}
		}
	}

}

func (self *state) firstSyncDone() {
	close(self.firstTa.closed)
	self.firstTa.done = 1
	self.bus.Publish(common.DwlDone)
}

func (self *state) bestPeer() p2p.Peer {
	var r p2p.Peer
	h := uint64(0)

	self.peers.Range(func(_, p interface{}) bool {
		p1 := p.(*syncPeer)
		s := p1.peer.GetState().(*handState)
		if s.S.Height > h {
			r = p1.peer
			h = s.S.Height
			return true
		}
		return true
	})

	return r
}
