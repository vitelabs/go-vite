package chain

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"github.com/vitelabs/go-vite/vm/quota"
	"github.com/vitelabs/go-vite/vm_db"
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

func (c *chain) GetAllRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressConsensusGroup)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed, snapshotHash is %s",
			snapshotHash))
		c.log.Error(cErr.Error(), "method", "GetAllRegisterList")
		return nil, cErr
	}

	// do something
	return abi.GetAllRegistrationList(sd, gid)
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

func (c *chain) GetPledgeBeneficialAmount(addr types.Address) (*big.Int, error) {

	sd, err := c.stateDB.NewStorageDatabase(c.GetLatestSnapshotBlock().Hash, types.AddressPledge)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed"))
		c.log.Error(cErr.Error(), "method", "GetPledgeBeneficialAmount")
		return nil, cErr
	}

	// do something
	return abi.GetPledgeBeneficialAmount(sd, addr)
}

// total
func (c *chain) GetPledgeQuota(addr types.Address) (*big.Int, *types.Quota, error) {

	amount, err := c.GetPledgeBeneficialAmount(addr)
	if err != nil {
		return nil, nil, err
	}

	vmDb := vm_db.NewVmDbByAddr(c, &addr)

	quota, err := quota.GetPledgeQuota(vmDb, addr, amount, c.GetLatestSnapshotBlock().Height)
	if err != nil {
		return nil, nil, err
	}
	return amount, &quota, nil
}

// total
func (c *chain) GetPledgeQuotas(addrList []types.Address) (map[types.Address]*types.Quota, error) {
	sb := c.GetLatestSnapshotBlock()
	sd, err := c.stateDB.NewStorageDatabase(sb.Hash, types.AddressPledge)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed"))
		c.log.Error(cErr.Error(), "method", "GetPledgeBeneficialAmount")
		return nil, cErr
	}

	quotaMap := make(map[types.Address]*types.Quota, len(addrList))

	for _, addr := range addrList {
		amount, err := abi.GetPledgeBeneficialAmount(sd, addr)
		if err != nil {
			cErr := errors.New(fmt.Sprintf("abi.GetPledgeBeneficialAmount failed, Addr is %s. Error: %s", addr, err))
			c.log.Error(cErr.Error(), "method", "GetPledgeQuotas")
			return nil, err
		}

		vmDb := vm_db.NewVmDbByAddr(c, &addr)
		quota, err := quota.GetPledgeQuota(vmDb, addr, amount, sb.Height)
		if err != nil {
			cErr := errors.New(fmt.Sprintf("quota.GetPledgeQuota failed. Error: %s", err))
			c.log.Error(cErr.Error(), "method", "GetPledgeQuotas")
			return nil, err
		}
		quotaMap[addr] = &quota
	}
	return quotaMap, nil

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

func (c *chain) GetAllTokenInfo() (map[types.TokenTypeId]*types.TokenInfo, error) {
	sd, err := c.stateDB.NewStorageDatabase(c.GetLatestSnapshotBlock().Hash, types.AddressMintage)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed"))
		c.log.Error(cErr.Error(), "method", "GetAllTokenInfo")

		return nil, cErr
	}
	return abi.GetTokenMap(sd)
}

func (c *chain) GetPledgeListByPage(snapshotHash types.Hash, lastKey []byte, count uint64) ([]*types.PledgeInfo, []byte, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressPledge)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed"))
		c.log.Error(cErr.Error(), "method", "GetPledgeAmountByPage")

		return nil, nil, cErr
	}
	return abi.GetPledgeListByPage(sd, lastKey, count)
}

func (c *chain) GetDexFundsByPage(snapshotHash types.Hash, lastAddress types.Address, count int) ([]*dex.UserFund, error) {
	sd, err := c.stateDB.NewStorageDatabase(snapshotHash, types.AddressDexFund)
	if err != nil {
		cErr := errors.New(fmt.Sprintf("c.stateDB.NewStorageDatabase failed"))
		c.log.Error(cErr.Error(), "method", "GetPledgeAmountByPage")
		return nil, cErr
	}
	return dex.GetUserFundsByPage(sd, lastAddress, count)
}