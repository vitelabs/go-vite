package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/quota"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
)

// TODO
func (c *chain) GetContractGidByAccountBlock(block *ledger.AccountBlock) (*types.Gid, error) {
	if block == nil {
		return nil, nil
	}

	return nil, nil
}

// TODO cache + inner contract
func (c *chain) GetContractGid(addr *types.Address) (*types.Gid, error) {
	account, getAccountErr := c.chainDb.Account.GetAccountByAddress(addr)
	if getAccountErr != nil {

		c.log.Error("Query account failed. Error is "+getAccountErr.Error(), "method", "GetContractGid")

		return nil, getAccountErr
	}

	gid, err := c.chainDb.Ac.GetContractGid(account.AccountId)

	if err != nil {
		c.log.Error("GetContractGid failed, error is "+err.Error(), "method", "GetContractGid")
		return nil, err
	}

	return gid, nil
}

func (c *chain) GetPledgeQuotas(snapshotHash types.Hash, beneficialList []types.Address) map[types.Address]uint64 {
	vmContext, err := vm_context.NewVmContext(c, &snapshotHash, nil, &contracts.AddressPledge)
	if err != nil {
		c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetPledgeQuotaList")
		return nil
	}

	quotas := make(map[types.Address]uint64)
	for _, addr := range beneficialList {
		quotas[addr] = quota.GetPledgeQuota(vmContext, addr)
	}
	return quotas
}
func (c *chain) GetPledgeQuota(snapshotHash types.Hash, beneficial types.Address) uint64 {
	vmContext, err := vm_context.NewVmContext(c, &snapshotHash, nil, &contracts.AddressPledge)
	if err != nil {
		c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetPledgeQuota")
		return 0
	}
	return quota.GetPledgeQuota(vmContext, beneficial)
}

func (c *chain) GetRegisterList(snapshotHash types.Hash, gid types.Gid) []*contracts.Registration {
	vmContext, err := vm_context.NewVmContext(c, &snapshotHash, nil, &contracts.AddressRegister)
	if err != nil {
		c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetRegisterList")
		return nil
	}
	return contracts.GetRegisterList(vmContext, gid)
}

func (c *chain) GetVoteMap(snapshotHash types.Hash, gid types.Gid) []*contracts.VoteInfo {
	vmContext, err := vm_context.NewVmContext(c, &snapshotHash, nil, &contracts.AddressVote)
	if err != nil {
		c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetVoteList")
		return nil
	}
	return contracts.GetVoteList(vmContext, gid)
}

func (c *chain) GetPledgeAmount(snapshotHash types.Hash, beneficial types.Address) *big.Int {
	vmContext, err := vm_context.NewVmContext(c, &snapshotHash, nil, &contracts.AddressPledge)
	if err != nil {
		c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetPledgeBeneficialAmount")
		return nil
	}
	return contracts.GetPledgeBeneficialAmount(vmContext, beneficial)
}

func (c *chain) GetConsensusGroupList(snapshotHash types.Hash) []*contracts.ConsensusGroupInfo {
	vmContext, err := vm_context.NewVmContext(c, &snapshotHash, nil, &contracts.AddressConsensusGroup)
	if err != nil {
		c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetActiveConsensusGroupList")
		return nil
	}
	return contracts.GetActiveConsensusGroupList(vmContext)
}

func (c *chain) GetBalanceList(snapshotHash types.Hash, tokenTypeId types.TokenTypeId, addressList []types.Address) map[types.Address]*big.Int {
	vmContext, err := vm_context.NewVmContext(c, &snapshotHash, nil, nil)
	if err != nil {
		c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetBalanceList")
		return nil
	}

	var balanceList map[types.Address]*big.Int
	for _, addr := range addressList {
		balanceList[addr] = vmContext.GetBalance(&addr, &tokenTypeId)
	}
	return balanceList
}

func (c *chain) GetTokenInfoById(tokenId *types.TokenTypeId) *contracts.TokenInfo {
	vmContext, err := vm_context.NewVmContext(c, nil, nil, &contracts.AddressMintage)
	if err != nil {
		c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetTokenInfoById")
		return nil
	}
	return contracts.GetTokenById(vmContext, *tokenId)
}
