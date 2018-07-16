package protocols

import (
	"github.com/vitelabs/go-vite/p2p"
	"sync"
	"github.com/vitelabs/go-vite/ledger"
	"fmt"
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
	pm.mutex.RLock()
	handler := pm.handlers[code]
	pm.mutex.RUnlock()

	if handler == nil {
		return fmt.Errorf("missing handler for msg code: %d\n", code)
	}

	return handler(s, peer)
}

func (pm *ProtocolManager) HandlePeer(peer *p2p.Peer) {
	protoPeer := &Peer{
		Peer: peer,
		ID: peer.ID().String(),
	}

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

			m.Deserialize(msg.Payload)
			if err := pm.HandleMsg(msg.Code, m, protoPeer); err != nil {
				peer.Errch <- err
			}
		}

	}
}

func NewProtocolManager(bc blockchain) *ProtocolManager {
	return &ProtocolManager {
		currentBlock: bc.GetLatestBlock(),
		ProtoHandlers: ProtoHandlers{
			handlers: make(map[uint64]ProtoHandler),
		},
	}
}
