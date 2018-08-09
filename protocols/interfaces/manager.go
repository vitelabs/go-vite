package interfaces

import protoType "github.com/vitelabs/go-vite/protocols/types"

type ProtocolManager interface {
	BestPeer() *protoType.Peer
	SendMsg(p *protoType.Peer, msg *protoType.Msg) error
	BroadcastMsg(msg *protoType.Msg) (fails []*protoType.Peer)
	Sync()
	SyncDone()
}
