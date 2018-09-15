package contracts

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm_context"
	"math/big"
)

var (
	AddressRegister, _       = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	AddressVote, _           = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2})
	AddressPledge, _         = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3})
	AddressConsensusGroup, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4})
	AddressMintage, _        = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5})
)

func GetTokenById(db vm_context.VmDatabase, tokenId types.TokenTypeId) *TokenInfo {
	data := db.GetStorage(&AddressMintage, helper.LeftPadBytes(tokenId.Bytes(), types.HashSize))
	if len(data) > 0 {
		tokenInfo := new(TokenInfo)
		ABI_mintage.UnpackVariable(tokenInfo, VariableNameMintage, data)
		return tokenInfo
	}
	return nil
}

func GetTokenMap(db vm_context.VmDatabase) map[types.TokenTypeId]*TokenInfo {
	iterator := db.NewStorageIterator(nil)
	tokenInfoMap := make(map[types.TokenTypeId]*TokenInfo)
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		tokenId, _ := types.BytesToTokenTypeId(key[types.HashSize-types.TokenTypeIdSize:])
		tokenInfo := new(TokenInfo)
		ABI_register.UnpackVariable(tokenInfo, VariableNameMintage, value)
		tokenInfoMap[tokenId] = tokenInfo
	}
	return tokenInfoMap
}

// get register address of gid
func GetRegisterList(db vm_context.VmDatabase, gid types.Gid) []*Registration {
	iterator := db.NewStorageIterator(gid.Bytes())
	registerList := make([]*Registration, 0)
	for {
		_, value, ok := iterator.Next()
		if !ok {
			break
		}
		registration := new(Registration)
		ABI_register.UnpackVariable(registration, VariableNameRegistration, value)
		if registration.Timestamp > 0 {
			registerList = append(registerList, registration)
		}
	}
	return registerList
}

// get voters of gid, return map<voter, super node>
func GetVoteMap(db vm_context.VmDatabase, gid types.Gid) []*VoteInfo {
	iterator := db.NewStorageIterator(gid.Bytes())
	voteInfoList := make([]*VoteInfo, 0)
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		voterAddr := GetAddrFromVoteKey(key)
		nodeName := new(string)
		ABI_vote.UnpackVariable(nodeName, VariableNameVoteStatus, value)
		voteInfoList = append(voteInfoList, &VoteInfo{voterAddr, *nodeName})
	}
	return voteInfoList
}

// get beneficial pledge amount
func GetPledgeAmount(db vm_context.VmDatabase, beneficial types.Address) *big.Int {
	locHash := types.DataHash(beneficial.Bytes()).Bytes()
	beneficialAmount := new(VariablePledgeBeneficial)
	err := ABI_pledge.UnpackVariable(beneficialAmount, VariableNamePledgeBeneficial, db.GetStorage(&AddressPledge, locHash))
	if err == nil {
		return beneficialAmount.Amount
	}
	return big.NewInt(0)
}

// get all consensus group info
func GetConsensusGroupList(db vm_context.VmDatabase) []*ConsensusGroupInfo {
	iterator := db.NewStorageIterator(nil)
	consensusGroupInfoList := make([]*ConsensusGroupInfo, 0)
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		consensusGroupInfo := new(ConsensusGroupInfo)
		ABI_pledge.UnpackVariable(consensusGroupInfo, VariableNameConsensusGroupInfo, value)
		consensusGroupInfo.Gid = GetGidFromConsensusGroupKey(key)
		consensusGroupInfoList = append(consensusGroupInfoList, consensusGroupInfo)
	}
	return consensusGroupInfoList
}

// get consensus group info by gid
func GetConsensusGroup(db vm_context.VmDatabase, gid types.Gid) *ConsensusGroupInfo {
	data := db.GetStorage(&AddressConsensusGroup, types.DataHash(gid.Bytes()).Bytes())
	if len(data) > 0 {
		consensusGroupInfo := new(ConsensusGroupInfo)
		ABI_pledge.UnpackVariable(consensusGroupInfo, VariableNameConsensusGroupInfo, data)
		consensusGroupInfo.Gid = gid
		return consensusGroupInfo
	}
	return nil
}
