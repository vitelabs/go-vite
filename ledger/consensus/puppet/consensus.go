package consensus_puppet

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/ledger/consensus"
)

type consensusPuppet struct {
	consensus.Consensus
}

func (consensusPuppet) VerifyAccountProducer(block *ledger.AccountBlock) (bool, error) {
	return true, nil
}
func (consensusPuppet) VerifyABsProducer(abs map[types.Gid][]*ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
	return nil, nil
}
func (consensusPuppet) VerifySnapshotProducer(block *ledger.SnapshotBlock) (bool, error) {
	return true, nil
}

func (puppet *consensusPuppet) Subscribe(gid types.Gid, id string, addr *types.Address, fn func(consensus.Event)) {

}
func (puppet *consensusPuppet) UnSubscribe(gid types.Gid, id string) {

}
func (puppet *consensusPuppet) SubscribeProducers(gid types.Gid, id string, fn func(event consensus.ProducersEvent)) {

}
