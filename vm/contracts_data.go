package vm

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

func GetTokenById(db VmDatabase, tokenId types.TokenTypeId) *types.TokenInfo {
	data := db.GetStorage(&AddressMintage, helper.LeftPadBytes(tokenId.Bytes(), types.HashSize))
	if len(data) > 0 {
		tokenInfo := new(types.TokenInfo)
		ABI_mintage.UnpackVariable(tokenInfo, VariableNameMintage, data)
		return tokenInfo
	}
	return nil
}

func GetTokenMap(db VmDatabase) map[types.TokenTypeId]*types.TokenInfo {
	iterator := db.NewStorageIterator(nil)
	tokenInfoMap := make(map[types.TokenTypeId]*types.TokenInfo)
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		tokenId, _ := types.BytesToTokenTypeId(key[types.HashSize-types.TokenTypeIdSize:])
		tokenInfo := new(types.TokenInfo)
		ABI_register.UnpackVariable(tokenInfo, VariableNameMintage, value)
		tokenInfoMap[tokenId] = tokenInfo
	}
	return tokenInfoMap
}

// get register address of gid
func GetRegisterList(db VmDatabase, gid types.Gid) []*types.Registration {
	iterator := db.NewStorageIterator(gid.Bytes())
	registerList := make([]*types.Registration, 0)
	for {
		_, value, ok := iterator.Next()
		if !ok {
			break
		}
		registration := new(types.Registration)
		ABI_register.UnpackVariable(registration, VariableNameRegistration, value)
		if registration.Timestamp > 0 {
			registerList = append(registerList, registration)
		}
	}
	return registerList
}

// get voters of gid, return map<voter, super node>
func GetVoteMap(db VmDatabase, gid types.Gid) []*types.VoteInfo {
	iterator := db.NewStorageIterator(gid.Bytes())
	voteInfoList := make([]*types.VoteInfo, 0)
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		voterAddr := getAddrFromVoteKey(key)
		nodeName := new(string)
		ABI_vote.UnpackVariable(nodeName, VariableNameVoteStatus, value)
		voteInfoList = append(voteInfoList, &types.VoteInfo{voterAddr, *nodeName})
	}
	return voteInfoList
}

// get beneficial pledge amount
func GetPledgeAmount(db VmDatabase, beneficial types.Address) *big.Int {
	locHash := types.DataHash(beneficial.Bytes()).Bytes()
	beneficialAmount := new(VariablePledgeBeneficial)
	err := ABI_pledge.UnpackVariable(beneficialAmount, VariableNamePledgeBeneficial, db.GetStorage(&AddressPledge, locHash))
	if err == nil {
		return beneficialAmount.Amount
	}
	return big.NewInt(0)
}

// get all consensus group info
func GetConsensusGroupList(db VmDatabase) []*types.ConsensusGroupInfo {
	iterator := db.NewStorageIterator(nil)
	consensusGroupInfoList := make([]*types.ConsensusGroupInfo, 0)
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		consensusGroupInfo := new(types.ConsensusGroupInfo)
		ABI_pledge.UnpackVariable(consensusGroupInfo, VariableNameConsensusGroupInfo, value)
		consensusGroupInfo.Gid = getGidFromConsensusGroupKey(key)
		consensusGroupInfoList = append(consensusGroupInfoList, consensusGroupInfo)
	}
	return consensusGroupInfoList
}

// get consensus group info by gid
func GetConsensusGroup(db VmDatabase, gid types.Gid) *types.ConsensusGroupInfo {
	data := db.GetStorage(&AddressConsensusGroup, types.DataHash(gid.Bytes()).Bytes())
	if len(data) > 0 {
		consensusGroupInfo := new(types.ConsensusGroupInfo)
		ABI_pledge.UnpackVariable(consensusGroupInfo, VariableNameConsensusGroupInfo, data)
		consensusGroupInfo.Gid = gid
		return consensusGroupInfo
	}
	return nil
}
