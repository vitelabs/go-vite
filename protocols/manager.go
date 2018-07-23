package protocols

import (
	"github.com/vitelabs/go-vite/p2p"
	"sync"
	"github.com/vitelabs/go-vite/ledger"
	"fmt"
	"log"
	"time"
	"math/big"
	protoType "github.com/vitelabs/go-vite/protocols/types"
	"github.com/vitelabs/go-vite/protocols/interfaces"
	ledgerHandler "github.com/vitelabs/go-vite/ledger/handler_interface"
)

var enoughPeersTimeout = 3 * time.Minute
const enoughPeers = 5
const broadcastConcurrency = 10

type ProtocolManager struct {
	Peers *PeersMap
	Start time.Time
	Syncing bool
	vite interfaces.Vite
	schain ledgerHandler.SnapshotChain
	achain ledgerHandler.AccountChain
	mutex sync.RWMutex
}

func (pm *ProtocolManager) HandleStatusMsg(status *protoType.StatusMsg, peer *protoType.Peer) {
	log.Printf("receive status from %s height %d \n", peer.ID, status.Height)

	peer.Update(status)
	pm.Sync()
}

func (pm *ProtocolManager) SendStatusMsg(peer *protoType.Peer) {
	// todo get genesis block hash
	currentBlock := pm.CurrentBlock()
	status := &protoType.StatusMsg{
		ProtocolVersion: protoType.Vite1,
		Height: currentBlock.Height,
		CurrentBlock: *currentBlock.Hash,
	}
	err := pm.SendMsg(peer, &protoType.Msg{
		Code: protoType.StatusMsgCode,
		Payload: status,
	})

	if err != nil {
		peer.Errch <- err
		log.Printf("send status msg to %s error: %v\n", peer.ID, err)
	} else {
		log.Printf("send status msg to %s, our height %d\n", peer.ID, status.Height)
	}
}

func (pm *ProtocolManager) HandlePeer(peer *p2p.Peer) {
	protoPeer := &protoType.Peer{
		Peer: peer,
		ID: peer.ID().String(),
	}
	pm.Peers.AddPeer(protoPeer)
	log.Printf("now wei have %d peers\n", pm.Peers.Count())

	go pm.SendStatusMsg(protoPeer)

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	var err error

	for {
		select {
		case <- peer.Closed:
			pm.Peers.DelPeer(protoPeer)
			return
		case msg := <- peer.ProtoMsg:
			switch msg.Code {
			case protoType.StatusMsgCode:
				m := new(protoType.StatusMsg)
				m.NetDeserialize(msg.Payload)
				go pm.HandleStatusMsg(m, protoPeer)
			case protoType.GetSnapshotBlocksMsgCode:
				m := new(protoType.GetSnapshotBlocksMsg)
				m.NetDeserialize(msg.Payload)
				err = pm.schain.HandleGetBlocks(m, protoPeer)
			case protoType.SnapshotBlocksMsgCode:
				m := new(protoType.SnapshotBlocksMsg)
				m.NetDeserialize(msg.Payload)
				err = pm.schain.HandleSendBlocks(m, protoPeer)
			case protoType.GetAccountBlocksMsgCode:
				m := new(protoType.GetAccountBlocksMsg)
				m.NetDeserialize(msg.Payload)
				err = pm.achain.HandleGetBlocks(m, protoPeer)
			case protoType.AccountBlocksMsgCode:
				m := new(protoType.AccountBlocksMsg)
				m.NetDeserialize(msg.Payload)
				err = pm.achain.HandleSendBlocks(m, protoPeer)
			default:
				peer.Errch <- fmt.Errorf("unknown message code %d from %s\n", msg.Code, peer.ID())
				return
			}

			if err != nil {
				log.Printf("pm handle msg %d from %s error: %v\n", msg.Code, peer.ID(), err)
				peer.Errch <- err
			} else {
				log.Printf("pm handle msg %d from %s done\n", msg.Code, peer.ID())
			}
		case <- ticker.C:
			go pm.SendStatusMsg(protoPeer)
		}
	}
}

func (pm *ProtocolManager) SendMsg(p *protoType.Peer, msg *protoType.Msg) error {
	payload, err := msg.Payload.NetSerialize()
	if err != nil {
		return fmt.Errorf("pm.SendMsg NetSerialize msg %d to %s error: %v\n", msg.Code, p.ID, err)
	}
	m := &p2p.Msg{
		Code: msg.Code,
		Payload: payload,
	}

	// broadcast to all peers
	if p == nil {
		pm.BroadcastMsg(msg)
		return nil
	}

	// send to the specified peer
	log.Printf("pm begin send msg %d to %s\n", msg.Code, p.ID)
	return p2p.Send(p.TS, m)
}

func (pm *ProtocolManager) BroadcastMsg(msg *protoType.Msg) (fails []*protoType.Peer) {
	log.Printf("pm broadcast msg %d\n", msg.Code)

	sent := make(map[*protoType.Peer]bool)

	pending := make(chan struct{}, broadcastConcurrency)

	broadcastPeer := func(p *protoType.Peer) {
		err := pm.SendMsg(p, msg)
		if err != nil {
			sent[p] = false
			log.Printf("pm broadcast msg %d to %s error: %v\n", msg.Code, p.ID, err)
		} else {
			log.Printf("pm broadcast msg %d to %s done\n", msg.Code, p.ID)
		}
		<- pending
	}

	for _, peer := range pm.Peers.peers {
		_, ok := sent[peer]
		if ok {
			continue
		}

		pending <- struct{}{}
		sent[peer] = true
		go broadcastPeer(peer)
	}

	for peer, ok := range sent {
		if !ok {
			fails = append(fails, peer)
		}
	}

	return fails
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
		if pm.schain.SyncPeer != nil {
			pm.mutex.Lock()
			if pm.Syncing {
				log.Println("is already syncing")
				pm.mutex.Unlock()
				return
			} else {
				pm.Syncing = true
				pm.mutex.Unlock()
			}

			pm.schain.SyncPeer(bestPeer)
			log.Printf("begin sync from %s to height %d\n", bestPeer.ID, bestPeer.Height.Uint64())
		} else {
			log.Println("missing sync method")
		}
	}
}

func (pm *ProtocolManager) SyncDone() {
	pm.mutex.Lock()
	pm.Syncing = false
	pm.mutex.Unlock()
}

func (pm *ProtocolManager) CurrentBlock() (block *ledger.SnapshotBlock) {
	block, err :=  pm.schain.GetLatestBlock()
	if err != nil {
		log.Fatalf("pm.chain.GetLatestBlock error: %v\n", err)
	}

	return block
}

func NewProtocolManager(vite interfaces.Vite) *ProtocolManager {
	return &ProtocolManager {
		Peers: NewPeersMap(),
		Start: time.Now(),
		schain: vite.Ledger().Sc(),
		achain: vite.Ledger().Ac(),
		vite: vite,
	}
}

// @section PeersMap
type PeersMap struct {
	peers map[string]*protoType.Peer
	rw sync.RWMutex
}

func NewPeersMap() *PeersMap {
	return &PeersMap{
		peers: make(map[string]*protoType.Peer),
	}
}

func (m *PeersMap) BestPeer() (best *protoType.Peer) {
	m.rw.RLock()
	defer m.rw.RUnlock()

	maxHeight := new(big.Int)
	for _, peer := range m.peers {
		fmt.Printf("peer: %#v, peer.Height: %#v\n", peer, peer.Height)
		cmp := peer.Height.Cmp(maxHeight)
		if cmp > 0 {
			maxHeight = peer.Height
			best = peer
		}
	}

	return
}

func (m *PeersMap) AddPeer(peer *protoType.Peer) {
	m.rw.Lock()
	m.peers[peer.ID] = peer
	m.rw.Unlock()
}

func (m *PeersMap) DelPeer(peer *protoType.Peer) {
	m.rw.Lock()
	delete(m.peers, peer.ID)
	m.rw.Unlock()
}

func (m *PeersMap) Count() int {
	m.rw.RLock()
	defer m.rw.RUnlock()

	return len(m.peers)
}
