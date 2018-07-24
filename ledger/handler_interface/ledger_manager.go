package handler_interface

type Manager interface {
	Ac() AccountChain
	Sc() SnapshotChain
}
