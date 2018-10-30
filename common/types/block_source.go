package types

type BlockSource uint16

const (
	Unkonwn         BlockSource = 0
	RemoteBroadcast             = 10
	RemoteFetch                 = 20
	Local                       = 30
	RollbackChain               = 40
	RemoteSync                  = 50
)
