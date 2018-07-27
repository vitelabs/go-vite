package handler_interface

type Manager interface {
	Ac() AccountChain
	Sc() SnapshotChain

	RegisterFirstSyncDown(chan<- int) // 0 represent success, not 0 represent failed.
}
