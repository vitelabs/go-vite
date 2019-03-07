package p2p

type MsgPool interface {
	receive(msg Msg) error
	send(msg Msg) error
	Start() error
	Stop() error
}

type peerPool struct {
	maxWorkers int
	protoMap   map[ProtocolID]Protocol
}

func (p *peerPool) receive(msg Msg) error {
	panic("implement me")
}

func (p *peerPool) send(msg Msg) error {
	panic("implement me")
}

func (p *peerPool) Start() error {

}

func (p *peerPool) Stop() error {

}
