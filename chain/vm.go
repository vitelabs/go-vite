package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

func (c *Chain) GetRegisterList(snapshotHash types.Hash, gid types.Gid) []types.Address {
	return nil
}

func (c *Chain) GetVoteMap(snapshotHash types.Hash, gid types.Gid) map[types.Address]types.Address {
	return nil
}

func (c *Chain) GetMortgageAmount(snapshotHash types.Hash, beneficial types.Address) *big.Int {
	return nil
}

func (c *Chain) GetConsensusGroupList(snapshotHash types.Hash) []*types.ConsensusGroupInfo {
	return nil
}
