package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

// sb height
func (c *chain) GetRegisterList(snapshotHash *types.Hash, gid *types.Gid) ([]*types.Registration, error) {
	ss, err := c.stateDB.NewSnapshotStorageIterator(snapshotHash, &types.AddressConsensusGroup, nil)
	if err != nil {
		c.log.Error(err.Error(), "method", "GetRegisterList")
		return nil, err
	}
	defer ss.Release()

	// do something

	return nil, nil
}

func (c *chain) GetVoteMap(snapshotHash *types.Hash, gid *types.Gid) ([]*types.VoteInfo, error) {
	ss, err := c.stateDB.NewSnapshotStorageIterator(snapshotHash, &types.AddressConsensusGroup, nil)
	if err != nil {
		c.log.Error(err.Error(), "method", "GetVoteMap")
		return nil, err
	}
	defer ss.Release()

	// do something

	return nil, nil
}

func (c *chain) GetPledgeAmount(snapshotHash *types.Hash, addr *types.Address) (*big.Int, error) {
	ss, err := c.stateDB.NewSnapshotStorageIterator(snapshotHash, &types.AddressConsensusGroup, nil)
	if err != nil {
		c.log.Error(err.Error(), "method", "GetPledgeAmount")
		return nil, err
	}
	defer ss.Release()

	// do something

	return nil, nil
}

// total
func (c *chain) GetPledgeQuota(snapshotHash *types.Hash, addr *types.Address) (uint64, error) {
	ss, err := c.stateDB.NewSnapshotStorageIterator(snapshotHash, &types.AddressPledge, nil)

	if err != nil {
		c.log.Error(err.Error(), "method", "GetPledgeQuota")
		return 0, err
	}
	defer ss.Release()

	// do something

	return 0, nil
}

// total
func (c *chain) GetPledgeQuotas(snapshotHash *types.Hash, addrList []*types.Address) (map[types.Address]uint64, error) {
	ss, err := c.stateDB.NewSnapshotStorageIterator(snapshotHash, &types.AddressPledge, nil)

	if err != nil {
		c.log.Error(err.Error(), "method", "GetPledgeQuotas")
		return nil, err
	}
	defer ss.Release()

	// do something

	return nil, nil
}

func (c *chain) GetTokenInfoById(tokenId *types.TokenTypeId) (*types.TokenInfo, error) {
	// do something
	c.stateDB.GetValue(&types.AddressMintage, tokenId.Bytes())
	return nil, nil
}

// TODO
func (c *chain) GetAllTokenInfo() ([]*types.TokenInfo, error) {
	// do something
	c.stateDB.NewStorageIterator(&types.AddressMintage, nil)
	return nil, nil
}
