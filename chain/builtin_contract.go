package chain

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"math/big"
)

// sb height
func (c *chain) GetRegisterList(snapshotHash *types.Hash, gid *types.Gid) ([]*types.Registration, error) {
	ss, err := c.getStateSnapshot(&types.AddressRegister, snapshotHash)
	if err != nil {
		c.log.Error(err.Error(), "method", "GetRegisterList")
		return nil, err
	}
	defer ss.Release()

	// do something
	ss.NewStorageIterator(nil)

	return nil, nil
}

func (c *chain) GetVoteMap(snapshotHash *types.Hash, gid *types.Gid) ([]*types.VoteInfo, error) {
	ss, err := c.getStateSnapshot(&types.AddressVote, snapshotHash)
	if err != nil {
		c.log.Error(err.Error(), "method", "GetVoteMap")
		return nil, err
	}
	defer ss.Release()

	// do something
	ss.NewStorageIterator(nil)
	return nil, nil
}

func (c *chain) GetPledgeAmount(snapshotHash *types.Hash, addr *types.Address) (*big.Int, error) {
	ss, err := c.getStateSnapshot(&types.AddressPledge, snapshotHash)
	if err != nil {
		c.log.Error(err.Error(), "method", "GetPledgeAmount")
		return nil, err
	}
	defer ss.Release()

	// do something
	ss.NewStorageIterator(nil)
	return nil, nil
}

// total
func (c *chain) GetPledgeQuota(snapshotHash *types.Hash, addr *types.Address) (uint64, error) {
	ss, err := c.getStateSnapshot(&types.AddressPledge, snapshotHash)
	if err != nil {
		c.log.Error(err.Error(), "method", "GetPledgeQuota")
		return 0, err
	}
	defer ss.Release()

	// do something
	ss.NewStorageIterator(nil)
	return 0, nil
}

// total
func (c *chain) GetPledgeQuotas(snapshotHash *types.Hash, addrList []*types.Address) (map[types.Address]uint64, error) {
	ss, err := c.getStateSnapshot(&types.AddressPledge, snapshotHash)
	if err != nil {
		c.log.Error(err.Error(), "method", "GetPledgeQuotas")
		return nil, err
	}
	defer ss.Release()

	// do something
	ss.NewStorageIterator(nil)
	return nil, nil
}

// TODO
func (c *chain) GetTokenInfoById(tokenId *types.TokenTypeId) (*types.TokenInfo, error) {
	c.stateDB.GetValue(&types.AddressMintage, tokenId.Bytes())
	return nil, nil
}

// TODO
func (c *chain) GetAllTokenInfo() ([]*types.TokenInfo, error) {
	c.stateDB.NewStorageIterator(&types.AddressMintage, nil)
	return nil, nil
}

func (c *chain) getStateSnapshot(addr *types.Address, snapshotHash *types.Hash) (interfaces.StateSnapshot, error) {
	ss, err := c.stateDB.NewStateSnapshot(addr, 1)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStateSnapshot failed, error is %s, snapshotHash is %s",
			err, snapshotHash))
		return nil, cErr
	}
	return ss, nil
}
