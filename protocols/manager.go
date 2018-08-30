package protocols

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/vitelabs/go-vite/ledger"
	ledgerHandler "github.com/vitelabs/go-vite/ledger/handler_interface"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/protocols/interfaces"
	protoType "github.com/vitelabs/go-vite/protocols/types"
	"math/big"
	"strconv"
	"sync"
	"time"
)

var enoughPeersTimeout = 1 * time.Minute

const enoughPeers = 2
const broadcastConcurrency = 10

type ProtocolManager struct {
	Peers    *peerSet
	Start    time.Time
	Syncing  bool
	vite     interfaces.Vite
	schain   ledgerHandler.SnapshotChain
	achain   ledgerHandler.AccountChain
	mutex    sync.RWMutex
	syncPeer *protoType.Peer
	log      log15.Logger
}

func (pm *ProtocolManager) HandleStatusMsg(status *protoType.StatusMsg, peer *protoType.Peer) {
	if status.ProtocolVersion != protoType.Vite1 {
		peer.Errch <- errors.New("protocol version not match")
		return
	}

	pm.log.Info("receive status msg", "from", peer.ID, "peerHeight", status.Height)

	peer.Update(status)
	// AddPeer after get status msg from it, ensure we get Height and Hash of peer.
	pm.Peers.Add(peer)
	pm.log.Info("now we have " + strconv.Itoa(pm.Peers.Count()) + " peers")

	// use goroutine to avoid block following msgs.
	go pm.Sync()
}

func (pm *ProtocolManager) SendStatusMsg(peer *protoType.Peer) {
	// todo: should get genesis block hash
	currentBlock := pm.CurrentBlock()
	status := &protoType.StatusMsg{
		ProtocolVersion: protoType.Vite1,
		Height:          currentBlock.Height,
		CurrentBlock:    *currentBlock.Hash,
	}
	err := pm.SendMsg(peer, &protoType.Msg{
		Code:    protoType.StatusMsgCode,
		Payload: status,
	})

	if err != nil {
		peer.Errch <- err
		pm.log.Error("send status msg error", "to", peer.ID, "error", err)
	} else {
		pm.log.Info("send status msg done", "to", peer.ID, "selfHeight", status.Height)
	}
}

func (pm *ProtocolManager) HandlePeer(peer *p2p.Peer) {
	protoPeer := &protoType.Peer{
		Peer:    peer,
		ID:      peer.ID().String(),
		Sending: make(chan struct{}, 1),
		Log:     log15.New("module", "pm/peer"),
	}

	// send status msg to peer synchronously.
	// ensure status msg is the first msg in this session.
	pm.CheckStatus(protoPeer)

	var err error

	for {
		select {
		case <-peer.Closed:
			pm.Peers.DelPeer(protoPeer)
			// if the syncing peer is disconnected, then begin Sync immediately.
			if protoPeer == pm.syncPeer {
				go pm.Sync()
			}
			return
		case msg := <-peer.ProtoMsg:
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
				err = pm.schain.HandleGetBlocks(m, protoPeer, msg.Id)
			case protoType.SnapshotBlocksMsgCode:
				m := new(protoType.SnapshotBlocksMsg)
				m.NetDeserialize(msg.Payload)
				err = pm.schain.HandleSendBlocks(m, protoPeer, msg.Id)
			case protoType.GetAccountBlocksMsgCode:
				m := new(protoType.GetAccountBlocksMsg)
				m.NetDeserialize(msg.Payload)
				err = pm.achain.HandleGetBlocks(m, protoPeer, msg.Id)
			case protoType.AccountBlocksMsgCode:
				m := new(protoType.AccountBlocksMsg)
				m.NetDeserialize(msg.Payload)
				err = pm.achain.HandleSendBlocks(m, protoPeer, msg.Id)
			default:
				peer.Errch <- fmt.Errorf("unknown message code %d from %s\n", msg.Code, peer.ID())
				return
			}

			if err != nil {
				pm.log.Error("pm handle msg error", "code", msg.Code, "from", peer.ID(), "error", err)
				peer.Errch <- err
			} else {
				pm.log.Info("pm handle msg done", "code", msg.Code, "from", peer.ID())
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
			case <-ticker.C:
				pm.SendStatusMsg(peer)
			case <-peer.Closed:
				return
			}
		}
	}()
}

func (pm *ProtocolManager) SendMsg(p *protoType.Peer, msg *protoType.Msg) error {
	var payload []byte
	var err error
	if msg.Payload != nil {
		payload, err = msg.Payload.NetSerialize()
	}

	if err != nil {
		return fmt.Errorf("pm.SendMsg NetSerialize msg %d to %s error: %v\n", msg.Code, p.ID, err)
	}
	m := &p2p.Msg{
		Code:    msg.Code,
		Id:      msg.Id,
		Payload: payload,
	}

	// broadcast to all peers
	if p == nil {
		pm.BroadcastMsg(msg)
		return nil
	}

	// send to the specified peer
	p.Sending <- struct{}{}
	pm.log.Info("pm begin send msg", "code", msg.Code, "to", p.ID, "size", len(payload))
	err = p2p.Send(p.TS, m)
	<-p.Sending
	return err
}

func (pm *ProtocolManager) BroadcastMsg(msg *protoType.Msg) (fails []*protoType.Peer) {
	pm.log.Info("pm broadcast msg", "code", msg.Code)

	sent := make(map[*protoType.Peer]bool)

	pending := make(chan struct{}, broadcastConcurrency)

	broadcastPeer := func(p *protoType.Peer) {
		err := pm.SendMsg(p, msg)
		if err != nil {
			sent[p] = false
			pm.log.Error("pm broadcast msg error", "code", msg.Code, "to", p.ID, "error", err)
		} else {
			pm.log.Info("pm broadcast msg done", "code", msg.Code, "to", p.ID)
		}
		<-pending
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

	bestPeer := pm.BestPeer()
	currentBlock := pm.CurrentBlock()

	if bestPeer != nil && bestPeer.Height.Cmp(currentBlock.Height) > 0 {
		if pm.schain.SyncPeer != nil {
			pm.mutex.Lock()
			if pm.Syncing {
				pm.log.Info("already syncing")
				pm.mutex.Unlock()
				return
			} else {
				pm.syncPeer = bestPeer
				pm.Syncing = true
				pm.mutex.Unlock()
			}

			pm.log.Info("begin sync", "from", bestPeer.ID, "to", bestPeer.Height.Uint64())
			pm.schain.SyncPeer(bestPeer)
		} else {
			pm.log.Info("missing sync handler")
		}
	} else {
		if bestPeer != nil && currentBlock != nil {
			pm.log.Info("no need sync from bestPeer", "peer", bestPeer.ID, "peerHeight", bestPeer.Height, "selfHeight", currentBlock.Height)
		} else {
			pm.log.Info("bestPeer or currentBlock is nil ")
		}
		// tell blockchain no need sync
		if pm.schain.SyncPeer != nil {
			pm.schain.SyncPeer(nil)
		}
	}
}

func (pm *ProtocolManager) BestPeer() *Peer {
	return pm.Peers.BestPeer()
}

func (pm *ProtocolManager) SyncDone() {
	pm.mutex.Lock()
	pm.Syncing = false
	pm.syncPeer = nil
	pm.mutex.Unlock()
}

func (pm *ProtocolManager) CurrentBlock() (block *ledger.SnapshotBlock) {
	block, err := pm.schain.GetLatestBlock()
	if err != nil {
		pm.log.Error("getLatestBlock error", "error", err)
	} else {
		pm.log.Info("getLatestBlock done", "hash", block.Hash, "height", block.Height)
	}

	return block
}

func NewProtocolManager(vite interfaces.Vite) *ProtocolManager {
	return &ProtocolManager{
		Peers:  NewPeerSet(),
		Start:  time.Now(),
		schain: vite.Ledger().Sc(),
		achain: vite.Ledger().Ac(),
		vite:   vite,
		log:    log15.New("module", "pm"),
	}
}
