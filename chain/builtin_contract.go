package chain

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"math/big"
)

// sb height
func (c *chain) GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressConsensusGroup)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed, snapshotHash is %s",
			snapshotHash))
		c.log.Error(cErr.Error(), "method", "GetRegisterList")
		return nil, cErr
	}

	// do something
	return abi.GetCandidateList(sd, gid)
}

func (c *chain) GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressConsensusGroup)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed, snapshotHash is %s",
			snapshotHash))
		c.log.Error(cErr.Error(), "method", "GetConsensusGroupList")
		return nil, cErr
	}

	// do something
	return abi.GetActiveConsensusGroupList(sd)
}

func (c *chain) GetVoteList(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressConsensusGroup)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed, snapshotHash is %s",
			snapshotHash))
		c.log.Error(cErr.Error(), "method", "GetVoteList")
		return nil, cErr
	}

	// do something
	return abi.GetVoteList(sd, gid)
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
	sd, err := c.stateDB.NewStorageDatabase(c.GetLatestSnapshotBlock().Hash, types.AddressMintage)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed"))
		c.log.Error(cErr.Error(), "method", "GetTokenInfoById")

		return nil, cErr
	}
	return abi.GetTokenById(sd, tokenId)
}

// TODO
func (c *chain) GetAllTokenInfo() (map[types.TokenTypeId]*types.TokenInfo, error) {
	// do something
	sd, err := c.stateDB.NewStorageDatabase(c.GetLatestSnapshotBlock().Hash, types.AddressMintage)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed"))
		c.log.Error(cErr.Error(), "method", "GetAllTokenInfo")

		return nil, cErr
	}
	return abi.GetTokenMap(sd)
}
