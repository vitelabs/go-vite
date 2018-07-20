package interfaces

type ProtocolManager interface {
	SendMsg(p *protoType.Peer, msg *protoType.Msg) error
	BroadcastMsg(msg *protoType.Msg) (fails []*protoType.Peer)
	Sync()
	SyncDone()
}
