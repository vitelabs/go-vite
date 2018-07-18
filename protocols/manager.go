package protocols

import (
	"github.com/vitelabs/go-vite/p2p"
	"sync"
	"github.com/vitelabs/go-vite/ledger"
	"fmt"
	"log"
	"errors"
)

type ProtoHandler func(Serializable, *Peer) error

type ProtoHandlers struct {
	mutex sync.RWMutex
	handlers map[uint64]ProtoHandler
	currentBlock ledger.SnapshotBlock
}

func (pm *ProtoHandlers) RegHandler(code uint64, fn ProtoHandler) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.handlers[code] = fn
}

type blockchain interface {
	GetLatestBlock() ledger.SnapshotBlock
}

type ProtocolManager struct {
	currentBlock ledger.SnapshotBlock
	ProtoHandlers
}

func (pm *ProtocolManager) HandleMsg(code uint64, s Serializable, peer *Peer) error {
	if code == StatusMsgCode {
		status, ok := s.(*StatusMsg)
		if ok {
			return pm.HandleStatusMsg(status, peer)
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

func (pm *ProtocolManager) HandleStatusMsg(status *StatusMsg, peer *Peer) error {
	peer.Version = int(status.ProtocolVersion)
	peer.Height = status.Height
	peer.Head = status.CurrentBlock
	ourHeight := pm.currentBlock.Height

	cmp := ourHeight.Cmp(peer.Height)
	if cmp < 0 {
		getblocks := &GetSnapshotBlocksMsg{
			Origin: *pm.currentBlock.Hash,
			Count: 20,
			Forward: true,
		}
		return pm.SendMsg(peer, &Msg{
			Code: GetSnapshotBlocksMsgCode,
			Payload: getblocks,
		})
	}

	return nil
	// todo: push block to peer, if our height greater than peers`
}

func (pm *ProtocolManager) HandlePeer(peer *p2p.Peer) {
	protoPeer := &Peer{
		Peer: peer,
		ID: peer.ID().String(),
	}

	go func() {
		// todo get genesis block hash
		status := &StatusMsg{
			ProtocolVersion: vite1,
			Height: pm.currentBlock.Height,
			CurrentBlock: *pm.currentBlock.Hash,
		}
		err := pm.SendMsg(protoPeer, &Msg{
			Code: StatusMsgCode,
			Payload: status,
		})

		if err != nil {
			peer.Errch <- err
			log.Printf("send status msg to %s error: %v\n", protoPeer.ID, err)
		} else {
			log.Printf("send status msg to %s\n", protoPeer.ID)
		}
	}()

	for {
		var m Serializable
		select {
		case <- peer.Closed:
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
				peer.Errch <- err
			}
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

	return p2p.Send(p.TS, m)
}

func NewProtocolManager(bc blockchain) *ProtocolManager {
	return &ProtocolManager {
		currentBlock: bc.GetLatestBlock(),
		ProtoHandlers: ProtoHandlers{
			handlers: make(map[uint64]ProtoHandler),
		},
	}
}
