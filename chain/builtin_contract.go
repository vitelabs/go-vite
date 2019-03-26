package chain

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

// sb height
func (c *chain) GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error) {
	ss, err := c.stateDB.NewSnapshotStorageIterator(&snapshotHash, &types.AddressConsensusGroup, nil)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewSnapshotStorageIterator failed, snapshotHash is %s",
			snapshotHash))
		return nil, cErr
	}

	if ss == nil {
		return nil, nil
	}

	defer ss.Release()

	// do something
	return nil, nil
}

func (c *chain) GetConsensusGroupList(snapshotHash types.Hash, addr types.Address) ([]types.Gid, error) {
	return nil, nil
}

func (c *chain) GetVoteMap(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error) {
	ss, err := c.stateDB.NewSnapshotStorageIterator(&snapshotHash, &types.AddressConsensusGroup, nil)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewSnapshotStorageIterator failed, snapshotHash is %s",
			snapshotHash))
		return nil, cErr
	}

	if ss == nil {
		return nil, nil
	}

	defer ss.Release()

	// do something

	return nil, nil
}

func (c *chain) GetPledgeAmount(addr types.Address) (*big.Int, error) {
	snapshotHash := c.GetLatestSnapshotBlock().Hash
	ss, err := c.stateDB.NewSnapshotStorageIterator(&c.GetLatestSnapshotBlock().Hash, &types.AddressPledge, nil)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewSnapshotStorageIterator failed, snapshotHash is %s",
			snapshotHash))
		return nil, cErr
	}

	if ss == nil {
		return nil, nil
	}

	defer ss.Release()

	// do something
	return nil, nil
}

// total
func (c *chain) GetPledgeQuota(addr types.Address) (*types.Quota, error) {
	snapshotHash := c.GetLatestSnapshotBlock().Hash

	ss, err := c.stateDB.NewSnapshotStorageIterator(&snapshotHash, &types.AddressPledge, nil)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewSnapshotStorageIterator failed, snapshotHash is %s",
			snapshotHash))
		return nil, cErr
	}

	if ss == nil {
		return nil, nil
	}
	defer ss.Release()

	// do something
	return nil, nil
}

// total
func (c *chain) GetPledgeQuotas(addrList []types.Address) (map[types.Address]*types.Quota, error) {
	snapshotHash := c.GetLatestSnapshotBlock().Hash

	ss, err := c.stateDB.NewSnapshotStorageIterator(&snapshotHash, &types.AddressPledge, nil)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewSnapshotStorageIterator failed, snapshotHash is %s",
			snapshotHash))
		return nil, cErr
	}

	if ss == nil {
		return nil, nil
	}

	defer ss.Release()

	// do something

	return nil, nil
}

func (c *chain) GetTokenInfoById(tokenId types.TokenTypeId) (*types.TokenInfo, error) {
	// do something
	c.stateDB.GetStorageValue(&types.AddressMintage, tokenId.Bytes())
	return nil, nil
}

// TODO
func (c *chain) GetAllTokenInfo() ([]*types.TokenInfo, error) {
	// do something
	c.stateDB.NewStorageIterator(&types.AddressMintage, nil)
	return nil, nil
}
