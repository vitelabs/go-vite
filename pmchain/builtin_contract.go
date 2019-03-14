package pmchain

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

func (c *chain) GetRegisterList(snapshotHash *types.Hash, gid *types.Gid) ([]*types.Registration, error) {
	return nil, nil
}

func (c *chain) GetVoteMap(snapshotHash *types.Hash, gid *types.Gid) ([]*types.VoteInfo, error) {
	return nil, nil
}

func (c *chain) GetPledgeAmount(snapshotHash *types.Hash, addr *types.Address) (*big.Int, error) {
	return nil, nil
}

// total
func (c *chain) GetPledgeQuota(snapshotHash *types.Hash, addr *types.Address) (uint64, error) {
	return 0, nil
}

// total
func (c *chain) GetPledgeQuotas(snapshotHash *types.Hash, addrList []*types.Address) (map[types.Address]uint64, error) {
	return nil, nil
}

func (c *chain) GetTokenInfoById(tokenId *types.TokenTypeId) (*types.TokenInfo, error) {
	return nil, nil
}

func (c *chain) GetAllTokenInfo() ([]*types.TokenInfo, error) {
	return nil, nil
}
