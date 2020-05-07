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

func (se *state) GetState() (interface{}, error) {
	return se.getHandState()
}

func (se *state) Handshake(peerId string, state []byte) error {
	msg := &handState{}
	err := json.Unmarshal(state, msg)
	if err != nil {
		return err
	}
	hs, err := se.getHandState()
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

func (se *state) getHandState() (*handState, error) {
	head, e := se.rw.HeadSnapshot()
	if e != nil {
		return nil, errors.New("get snapshot head block fail.")

	}
	genesis, e := se.rw.GenesisSnapshot()
	if e != nil {
		return nil, errors.New("get genesis block fail.")
	}
	msg := &handState{GenesisHash: genesis.Hash(), S: peerState{Height: head.Height(), Hash: head.Hash()}}
	return msg, nil
}

func (se *state) DecodeState(state []byte) interface{} {
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

func (se *state) EncodeState(state interface{}) []byte {
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

func (se *state) update(msg *stateMsg, peer p2p.Peer) {
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
	head, e := se.rw.HeadSnapshot()
	if e != nil {
		log.Error("read snapshot head error:%v", e)
		return
	}
	if msg.Height > head.Height() {
		se.fetcher.fetchSnapshotBlockFromPeer(common.HashHeight{Hash: msg.Hash, Height: msg.Height}, peer)
	}
}
func (se *state) peerConnected(peer p2p.Peer) {
	se.peers.Store(peer.Id(), &syncPeer{peer: peer})
}
func (se *state) peerClosed(peer p2p.Peer) {
	se.peers.Delete(peer.Id())
}
func (se *state) start() {
	go se.loop()
	go se.syncFirst()
}

func (se *state) loop() {
	se.wg.Add(1)
	defer se.wg.Done()
	ticker := time.NewTicker(time.Second * 20)
	for {
		select {
		case <-se.closed:
			return
		case <-ticker.C:
			head, e := se.rw.HeadSnapshot()
			if e != nil {
				log.Error("read snapshot head error:%v", e)
				continue
			}
			stateMsg := stateMsg{Hash: head.Hash(), Height: head.Height()}
			log.Info("sync state, state:%v", stateMsg)
			se.sender.broadcastState(stateMsg)
		}
	}
}

func (se *state) syncDone() bool {
	return se.firstTa.done > 0
}

func (se *state) stop() {
	close(se.closed)
	se.wg.Wait()
}
func (se *state) syncFirst() {
	se.wg.Add(1)
	defer se.wg.Done()
	ta := se.firstTa

	t := time.NewTicker(time.Second * 2)

	timeout := time.NewTicker(time.Second * 50)
NET:
	for {
		select {
		case <-se.closed:
			return
		case <-ta.closed:
			return
		case <-timeout.C:
			log.Info("link other node fail. but started.")
			se.firstSyncDone()
		case <-t.C:
			i := 0
			se.peers.Range(func(_, _ interface{}) bool {
				i++
				return true
			})
			if i > 0 {
				p := se.bestPeer()
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

	head, e := se.rw.HeadSnapshot()
	if e != nil {
		log.Error("read snapshot head error:%v", e)
		return
	}
	if head.Height() >= ta.height {
		log.Info("need not sync, self height:%d, peer height:%d, peer id:%s", head.Height(), ta.height, ta.peer.Id())
		se.firstSyncDone()
		return
	}
	se.sender.requestSnapshotBlockByPeer(common.HashHeight{Hash: ta.hash, Height: ta.height}, ta.peer)

	for {
		select {
		case <-se.closed:
			return
		case <-ta.closed:
			return
		case <-t.C:
			head, e := se.rw.HeadSnapshot()
			if e != nil {
				log.Error("read snapshot head error:%v", e)
				continue
			}
			if head.Height() > ta.height {
				log.Info("sync first finish.")
				se.firstSyncDone()
				return
			}
		}
	}

}

func (se *state) firstSyncDone() {
	close(se.firstTa.closed)
	se.firstTa.done = 1
	se.bus.Publish(common.DwlDone)
}

func (se *state) bestPeer() p2p.Peer {
	var r p2p.Peer
	h := uint64(0)

	se.peers.Range(func(_, p interface{}) bool {
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
