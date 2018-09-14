package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
)

func GetTokenById(db VmDatabase, tokenId types.TokenTypeId) *TokenInfo {
	data := db.GetStorage(&AddressMintage, util.LeftPadBytes(tokenId.Bytes(), types.HashSize))
	if len(data) > 0 {
		tokenInfo := new(TokenInfo)
		ABI_mintage.UnpackVariable(tokenInfo, VariableNameMintage, data)
		return tokenInfo
	}
	return nil
}

func GetTokenMap(db VmDatabase) map[types.TokenTypeId]*TokenInfo {
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
func GetRegisterList(db VmDatabase, gid types.Gid) []*VariableRegistration {
	iterator := db.NewStorageIterator(gid.Bytes())
	registerList := make([]*VariableRegistration, 0)
	for {
		_, value, ok := iterator.Next()
		if !ok {
			break
		}
		registration := new(VariableRegistration)
		ABI_register.UnpackVariable(registration, VariableNameRegistration, value)
		if registration.Timestamp > 0 {
			registerList = append(registerList, registration)
		}
	}
	return registerList
}

// get voters of gid, return map<voter, super node>
func GetVoteMap(db VmDatabase, gid types.Gid) []*VoteInfo {
	iterator := db.NewStorageIterator(gid.Bytes())
	voteInfoList := make([]*VoteInfo, 0)
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		voterAddr := getAddrFromVoteKey(key)
		nodeName := new(string)
		ABI_vote.UnpackVariable(nodeName, VariableNameVoteStatus, value)
		voteInfoList = append(voteInfoList, &VoteInfo{voterAddr, *nodeName})
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
func GetConsensusGroupList(db VmDatabase) []*ConsensusGroupInfo {
	iterator := db.NewStorageIterator(nil)
	consensusGroupInfoList := make([]*ConsensusGroupInfo, 0)
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		variable := new(VariableConsensusGroupInfo)
		ABI_pledge.UnpackVariable(variable, VariableNameConsensusGroupInfo, value)
		consensusGroupInfoList = append(consensusGroupInfoList, &ConsensusGroupInfo{*variable, getGidFromConsensusGroupKey(key)})
	}
	return consensusGroupInfoList
}

// get consensus group info by gid
func GetConsensusGroup(db VmDatabase, gid types.Gid) *ConsensusGroupInfo {
	data := db.GetStorage(&AddressConsensusGroup, types.DataHash(gid.Bytes()).Bytes())
	if len(data) > 0 {
		variable := new(VariableConsensusGroupInfo)
		ABI_pledge.UnpackVariable(variable, VariableNameConsensusGroupInfo, data)
		return &ConsensusGroupInfo{*variable, gid}
	}
	return nil
}
