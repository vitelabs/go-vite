package chain

import (
	"bytes"
	"math/big"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/quota"
	"github.com/vitelabs/go-vite/vm_context"
)

func (c *chain) GetContractGidByAccountBlock(block *ledger.AccountBlock) (*types.Gid, error) {
	if block == nil {
		return nil, nil
	}

	return c.GetContractGid(&block.AccountAddress)
}

// TODO cache
func (c *chain) GetContractGid(addr *types.Address) (*types.Gid, error) {
	if addr == nil {
		return nil, nil
	}

	if bytes.Equal(addr.Bytes(), contracts.AddressRegister.Bytes()) ||
		bytes.Equal(addr.Bytes(), contracts.AddressVote.Bytes()) ||
		bytes.Equal(addr.Bytes(), contracts.AddressPledge.Bytes()) ||
		bytes.Equal(addr.Bytes(), contracts.AddressConsensusGroup.Bytes()) ||
		bytes.Equal(addr.Bytes(), contracts.AddressMintage.Bytes()) ||
		bytes.Equal(addr.Bytes(), contracts.AddressDexFund.Bytes()) ||
		bytes.Equal(addr.Bytes(), contracts.AddressDexTrade.Bytes()) {
		return &types.DELEGATE_GID, nil
	}

	account, getAccountErr := c.chainDb.Account.GetAccountByAddress(addr)
	if getAccountErr != nil {

		c.log.Error("Query account failed. Error is "+getAccountErr.Error(), "method", "GetContractGid")

		return nil, getAccountErr
	}
	if account == nil {
		return nil, nil
	}

	gid, err := c.chainDb.Ac.GetContractGid(account.AccountId)

	if err != nil {
		c.log.Error("GetContractGid failed, error is "+err.Error(), "method", "GetContractGid")
		return nil, err
	}

	return gid, nil
}

func (c *chain) GetPledgeQuotas(snapshotHash types.Hash, beneficialList []types.Address) map[types.Address]uint64 {
	quotas := make(map[types.Address]uint64)
	for _, addr := range beneficialList {
		vmContext, err := vm_context.NewVmContext(c, &snapshotHash, nil, &addr)
		if err != nil {
			c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetPledgeQuotaList")
			return nil
		}
		quotas[addr] = quota.GetPledgeQuota(vmContext, addr, c.GetPledgeAmount(snapshotHash, addr))
	}
	return quotas
}
func (c *chain) GetPledgeQuota(snapshotHash types.Hash, beneficial types.Address) uint64 {
	vmContext, err := vm_context.NewVmContext(c, &snapshotHash, nil, &beneficial)
	if err != nil {
		c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetPledgeQuota")
		return 0
	}
	return quota.GetPledgeQuota(vmContext, beneficial, c.GetPledgeAmount(snapshotHash, beneficial))
}

func (c *chain) GetRegisterList(snapshotHash types.Hash, gid types.Gid) []*contracts.Registration {
	vmContext, err := vm_context.NewVmContext(c, &snapshotHash, nil, nil)
	if err != nil {
		c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetRegisterList")
		return nil
	}
	return contracts.GetRegisterList(vmContext, gid)
}

func (c *chain) GetVoteMap(snapshotHash types.Hash, gid types.Gid) []*contracts.VoteInfo {
	vmContext, err := vm_context.NewVmContext(c, &snapshotHash, nil, nil)
	if err != nil {
		c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetVoteList")
		return nil
	}
	return contracts.GetVoteList(vmContext, gid)
}

func (c *chain) GetPledgeAmount(snapshotHash types.Hash, beneficial types.Address) *big.Int {
	vmContext, err := vm_context.NewVmContext(c, &snapshotHash, nil, nil)
	if err != nil {
		c.log.Error("NewVmContext failed, error is "+err.Error(), "method", "GetPledgeBeneficialAmount")
		return nil
	}
	return contracts.GetPledgeBeneficialAmount(vmContext, beneficial)
}

func (c *chain) GetConsensusGroupList(snapshotHash types.Hash) []*contracts.ConsensusGroupInfo {
	vmContext, err := vm_context.NewVmContext(c, &snapshotHash, nil, nil)
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
