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

var enoughPeersTimeout = 1 * time.Minute
const enoughPeers = 2
const broadcastConcurrency = 10

type ProtocolManager struct {
	Peers *PeersMap
	Start time.Time
	Syncing bool
	vite interfaces.Vite
	schain ledgerHandler.SnapshotChain
	achain ledgerHandler.AccountChain
	mutex sync.RWMutex
	syncPeer *protoType.Peer
}

func (pm *ProtocolManager) HandleStatusMsg(status *protoType.StatusMsg, peer *protoType.Peer) {
	log.Printf("receive status from %s height %d \n", peer.ID, status.Height)

	peer.Update(status)
	// AddPeer after get status msg from it, ensure we get Height and Hash of peer.
	pm.Peers.AddPeer(peer)
	log.Printf("now we have %d peers\n", pm.Peers.Count())

	// use goroutine to avoid block following msgs.
	go pm.Sync()
}

func (pm *ProtocolManager) SendStatusMsg(peer *protoType.Peer) {
	// todo: should get genesis block hash
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
		Sending: make(chan struct{}, 1),
	}

	// send status msg to peer synchronously.
	// ensure status msg is the first msg in this session.
	pm.CheckStatus(protoPeer)

	var err error

	for {
		select {
		case <- peer.Closed:
			pm.Peers.DelPeer(protoPeer)
			// if the syncing peer is disconnected, then begin Sync immediately.
			if protoPeer == pm.syncPeer {
				go pm.Sync()
			}
			return
		case msg := <- peer.ProtoMsg:
			switch msg.Code {
			case protoType.StatusMsgCode:
				m := new(protoType.StatusMsg)
				m.NetDeserialize(msg.Payload)
				// don`t use goroutine handle status msg.
				// ensure status msg is handled before other msg.
				// get Height of peer correctly.
				pm.HandleStatusMsg(m, protoPeer)
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
		}
	}
}

func (pm *ProtocolManager) CheckStatus(peer *protoType.Peer) {
	pm.SendStatusMsg(peer)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <- ticker.C:
				pm.SendStatusMsg(peer)
			case <- peer.Closed:
				return
			}
		}
	}()
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
	p.Sending <- struct{}{}
	log.Printf("pm begin send msg %d to %s %d bytes\n", msg.Code, p.ID, len(payload))
	err = p2p.Send(p.TS, m)
	<- p.Sending
	return err
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
				log.Println("already syncing")
				pm.mutex.Unlock()
				return
			} else {
				pm.syncPeer = bestPeer
				pm.Syncing = true
				pm.mutex.Unlock()
			}

			log.Printf("begin sync from %s until height %d\n", bestPeer.ID, bestPeer.Height.Uint64())
			pm.schain.SyncPeer(bestPeer)
		} else {
			log.Println("missing sync method")
		}
	} else {
		log.Printf("no need sync from bestPeer %s\n at height %d, self Height %d\n", bestPeer.ID, bestPeer.Height, currentBlock.Height)
		// tell blockchain no need sync
		if pm.schain.SyncPeer != nil {
			pm.schain.SyncPeer(nil)
		}
	}
}

func (pm *ProtocolManager) SyncDone() {
	pm.mutex.Lock()
	pm.Syncing = false
	pm.syncPeer = nil
	pm.mutex.Unlock()
}

func (pm *ProtocolManager) CurrentBlock() (block *ledger.SnapshotBlock) {
	block, err :=  pm.schain.GetLatestBlock()
	if err != nil {
		log.Fatalf("pm.chain.GetLatestBlock error: %v\n", err)
	} else {
		log.Printf("self latestblock: %s at height %d\n", block.Hash, block.Height)
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
