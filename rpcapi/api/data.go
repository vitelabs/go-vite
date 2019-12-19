package api

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	apidex "github.com/vitelabs/go-vite/rpcapi/api/dex"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"math/big"
)

type DataApi struct {
	chain chain.Chain
	log   log15.Logger
}

func NewDataApi(vite *vite.Vite) *DataApi {
	return &DataApi{
		chain: vite.Chain(),
		log:   log15.New("module", "rpc_api/data_api"),
	}
}

func (p DataApi) String() string {
	return "DataApi"
}

type GetPledgeListByPageResult struct {
	PledgeInfoList []*types.StakeInfo `json:"list"`
	LastKey        string             `json:"lastKey"`
}

func (p *DataApi) GetPledgeListByPage(snapshotHash types.Hash, lastKey string, count uint64) (*GetPledgeListByPageResult, error) {
	lastKeyBytes, err := hex.DecodeString(lastKey)
	if err != nil {
		return nil, err
	}
	list, lastKeyBytes, err := p.chain.GetStakeListByPage(snapshotHash, lastKeyBytes, count)
	if err != nil {
		return nil, err
	}
	return &GetPledgeListByPageResult{list, hex.EncodeToString(lastKeyBytes)}, nil
}

func (p DataApi) GetDexUserFundsByPage(snapshotHash types.Hash, lastAddress string, count int) (*apidex.Funds, error) {
	if count <= 0 {
		return nil, dex.InvalidInputParamErr
	}
	var lastAddr = types.ZERO_ADDRESS
	if len(lastAddress) > 0 {
		if addr, err := types.HexToAddress(lastAddress); err != nil {
			return nil, err
		} else {
			lastAddr = addr
		}
	}
	if funds, err := p.chain.GetDexFundsByPage(snapshotHash, lastAddr, count); err != nil {
		return nil, err
	} else {
		fundsRes := &apidex.Funds{}
		for _, fund := range funds {
			simpleFund := &apidex.SimpleFund{}
			if address, err := types.BytesToAddress(fund.Address); err != nil {
				return nil, err
			} else {
				simpleFund.Address = address.String()
			}
			for _, acc := range fund.Accounts {
				simpleAcc := &apidex.SimpleAccountInfo{}
				if token, err := types.BytesToTokenTypeId(acc.Token); err != nil {
					return nil, err
				} else {
					simpleAcc.Token = token.String()
				}
				if len(acc.Available) > 0 {
					simpleAcc.Available = apidex.AmountBytesToString(acc.Available)
				}
				if len(acc.Locked) > 0 {
					simpleAcc.Locked = apidex.AmountBytesToString(acc.Locked)
				}
				simpleFund.Accounts = append(simpleFund.Accounts, simpleAcc)
			}
			fundsRes.Funds = append(fundsRes.Funds, simpleFund)
		}
		return fundsRes, nil
	}
}

func (p DataApi) GetDexPledgeListByPage(snapshotHash types.Hash, lastKey string, count int) (*GetPledgeListByPageResult, error) {
	lastKeyBytes, err := hex.DecodeString(lastKey)
	if err != nil {
		return nil, err
	}
	list, lastKeyBytes, err := p.chain.GetDexStakeListByPage(snapshotHash, lastKeyBytes, count)
	if err != nil {
		return nil, err
	}
	var plist = make([]*types.StakeInfo, len(list))
	if len(list) > 0 {
		for i, info := range list {
			pInfo := &types.StakeInfo{}
			pInfo.Amount = new(big.Int).SetBytes(info.Amount)
			pInfo.Beneficiary = types.AddressDexFund
			pInfo.IsDelegated = true
			pInfo.DelegateAddress = types.AddressDexFund
			pInfo.Bid = uint8(info.StakeType)
			pInfo.StakeAddress, _ = types.BytesToAddress(info.Address)
			plist[i] = pInfo
		}
	}
	return &GetPledgeListByPageResult{plist, hex.EncodeToString(lastKeyBytes)}, nil
}
