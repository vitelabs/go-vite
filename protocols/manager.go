package protocols

import (
	"github.com/vitelabs/go-vite/p2p"
	"sync"
	"fmt"
	"github.com/vitelabs/go-vite/ledger"
)

type ProtoHandler func(Serializable, p2p.Peer) error

type ProtoMap struct {
	mutex sync.Mutex
	handlers map[uint64]ProtoHandler
	currentBlock ledger.SnapshotBlock

}

func (pm *ProtoMap) RegHandler(code uint64, fn ProtoHandler) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.handlers[code] = fn
}

func (pm *ProtoMap) HandleMsg(code uint64, s Serializable, peer p2p.Peer) error {
	handler := pm.handlers[code]
	if handler == nil {
		return fmt.Errorf("missing handler for msg code: %d\n", code)
	}

	return handler(s, peer)
}

func (pm *ProtoMap) HandlePeer(peer p2p.Peer) {
	// todo create protocol Peer

	for {
		var m Serializable
		select {
		case <- peer.Closed:
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
			if err := pm.HandleMsg(msg.Code, m, peer); err != nil {
				peer.Errch <- err
			}
		}

	}
}

type blockchain interface {
	GetLatestBlock() ledger.SnapshotBlock
}

type ProtocolManager struct {
	currentBlock ledger.SnapshotBlock
	ProtoMap
}

func NewProtocolManager(bc blockchain) *ProtocolManager {
	return &ProtocolManager {
		currentBlock: bc.GetLatestBlock(),
		ProtoMap: ProtoMap{
			handlers: make(map[uint64]ProtoHandler),
		},
	}
}
