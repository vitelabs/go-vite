package types

type BlockSource uint16

const (
	Unkonwn         BlockSource = 0
	RemoteBroadcast             = 1
	RemoteFetch                 = 2
	Local                       = 3
	RollbackChain               = 4
)
