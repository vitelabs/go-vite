package protocols

import (
	"github.com/vitelabs/go-vite/p2p"
	"sync"
	"github.com/vitelabs/go-vite/ledger"
	"fmt"
	"log"
	"errors"
	"time"
	"math/big"
)

var enoughPeersTimeout = 3 * time.Minute
const enoughPeers = 5

type ProtoHandler func(Serializable, *Peer) error
type SyncPeer func(*Peer)

type ProtoHandlers struct {
	mutex sync.RWMutex
	handlers map[uint64]ProtoHandler
	sync SyncPeer
}

func (pm *ProtoHandlers) RegHandler(code uint64, fn ProtoHandler) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.handlers[code] = fn
}

func (pm *ProtoHandlers) RegSync(fn SyncPeer) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.sync = fn
}

type blockchain interface {
	GetLatestBlock() (*ledger.SnapshotBlock, error)
}

type ProtocolManager struct {
	ProtoHandlers
	Peers *PeersMap
	Start time.Time
	Syncing bool
	chain blockchain
}

func (pm *ProtocolManager) HandleMsg(code uint64, s Serializable, peer *Peer) error {
	if code == StatusMsgCode {
		status, ok := s.(*StatusMsg)
		if ok {
			pm.HandleStatusMsg(status, peer)
			return nil
		} else {
			return errors.New("status msg payload unmatched")
		}
	}

	pm.mutex.RLock()
	handler := pm.handlers[code]
	pm.mutex.RUnlock()

	if handler == nil {
		return fmt.Errorf("missing handler for msg code: %d\n", code)
	}

	return handler(s, peer)
}

func (pm *ProtocolManager) HandleStatusMsg(status *StatusMsg, peer *Peer) {
	log.Printf("receive status from %s height %d \n", peer.ID, status.Height)

	peer.Update(status)
	pm.Sync()
}

func (pm *ProtocolManager) SendStatusMsg(peer *Peer) {
	// todo get genesis block hash
	currentBlock := pm.CurrentBlock()
	status := &StatusMsg{
		ProtocolVersion: vite1,
		Height: currentBlock.Height,
		CurrentBlock: *currentBlock.Hash,
	}
	err := pm.SendMsg(peer, &Msg{
		Code: StatusMsgCode,
		Payload: status,
	})

	if err != nil {
		peer.Errch <- err
		log.Printf("send status msg to %s error: %v\n", peer.ID, err)
	} else {
		log.Printf("send status msg to %s\n", peer.ID)
	}
}

func (pm *ProtocolManager) HandlePeer(peer *p2p.Peer) {
	protoPeer := &Peer{
		Peer: peer,
		ID: peer.ID().String(),
	}
	pm.Peers.AddPeer(protoPeer)

	go pm.SendStatusMsg(protoPeer)

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		var m Serializable
		select {
		case <- peer.Closed:
			pm.Peers.DelPeer(protoPeer)
			return
		case msg := <- peer.ProtoMsg:
			switch msg.Code {
			case StatusMsgCode:
				m = new(StatusMsg)
			case GetSnapshotBlocksMsgCode:
				m = new(GetSnapshotBlocksMsg)
			case SnapshotBlocksMsgCode:
				m = new(SnapshotBlocksMsg)
			case GetAccountBlocksMsgCode:
				m = new(GetAccountBlocksMsg)
			case AccountBlocksMsgCode:
				m = new(AccountBlocksMsg)
			default:
				peer.Errch <- fmt.Errorf("unknown message code %d\n", msg.Code)
				return
			}

			m.NetDeserialize(msg.Payload)

			if err := pm.HandleMsg(msg.Code, m, protoPeer); err != nil {
				log.Printf("pm handle msg %d error: %v\n", msg.Code, err)
				peer.Errch <- err
			} else {
				log.Printf("pm handle msg %d\n", msg.Code)
			}
		case <- ticker.C:
			go pm.SendStatusMsg(protoPeer)
		}
	}
}

func (pm *ProtocolManager) SendMsg(p *Peer, msg *Msg) error {
	payload, err := msg.Payload.NetSerialize()
	if err != nil {
		return fmt.Errorf("protocolManager Send error: %v\n", err)
	}
	m := &p2p.Msg{
		Code: msg.Code,
		Payload: payload,
	}

	// broadcast to all peers
	if p == nil {
		for _, peer := range pm.Peers.peers {
			go p2p.Send(peer.TS, m)
		}

		log.Printf("pm broadcast msg %d\n", msg.Code)
		return nil
	}

	// send to the specified peer
	log.Printf("pm send msg %d to %s\n", msg.Code, p.ID)
	return p2p.Send(p.TS, m)
}

func (pm *ProtocolManager) Sync() {
	pm.mutex.RLock()
	if pm.Syncing {
		pm.mutex.RUnlock()
		return
	}
	pm.mutex.RUnlock()

	if pm.Peers.Count() < enoughPeers {
		wait := time.Now().Sub(pm.Start)
		if wait < enoughPeersTimeout {
			time.Sleep(enoughPeersTimeout - wait)
		}
	}

	bestPeer := pm.Peers.BestPeer()
	currentBlock := pm.CurrentBlock()
	if bestPeer.Height.Cmp(currentBlock.Height) > 0 {
		pm.mutex.Lock()
		if pm.Syncing {
			// do nothing.
		} else if pm.sync != nil {
			pm.Syncing = true
			pm.sync(bestPeer)
			log.Printf("begin sync from %s to height \n", bestPeer.ID, bestPeer.Height)
		} else {
			log.Println("missing sync method")
		}
		pm.mutex.Unlock()
	}
}

func (pm *ProtocolManager) SyncDone() {
	pm.mutex.Lock()
	pm.Syncing = false
	pm.mutex.Unlock()
}

func (pm *ProtocolManager) CurrentBlock() (block *ledger.SnapshotBlock) {
	block, err :=  pm.chain.GetLatestBlock()
	if err != nil {
		log.Fatalf("pm.chain.GetLatestBlock error: %v\n", err)
	}

	return block
}

func NewProtocolManager(bc blockchain) *ProtocolManager {
	return &ProtocolManager {
		ProtoHandlers: ProtoHandlers{
			handlers: make(map[uint64]ProtoHandler),
		},
		Peers: NewPeersMap(),
		Start: time.Now(),
		chain: bc,
	}
}

// @section PeersMap
type PeersMap struct {
	peers map[string]*Peer
	rw sync.RWMutex
}

func NewPeersMap() *PeersMap {
	return &PeersMap{
		peers: make(map[string]*Peer),
	}
}

func (m *PeersMap) BestPeer() (best *Peer) {
	m.rw.RLock()
	defer m.rw.RUnlock()

	maxHeight := new(big.Int)
	for _, peer := range m.peers {
		cmp := peer.Height.Cmp(maxHeight)
		if cmp > 0 {
			maxHeight = peer.Height
			best = peer
		}
	}

	return
}

func (m *PeersMap) AddPeer(peer *Peer) {
	m.rw.Lock()
	m.peers[peer.ID] = peer
	m.rw.Unlock()
}

func (m *PeersMap) DelPeer(peer *Peer) {
	m.rw.Lock()
	delete(m.peers, peer.ID)
	m.rw.Unlock()
}

func (m *PeersMap) Count() int {
	m.rw.RLock()
	defer m.rw.RUnlock()

	return len(m.peers)
}
