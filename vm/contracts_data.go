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

func GetTokenList(db VmDatabase) []*TokenInfo {
	// TODO
	return nil
}

// get register address of gid
func GetRegisterList(db VmDatabase, gid types.Gid) []types.Address {
	iterator := db.NewStorageIterator(gid.Bytes())
	registerList := make([]types.Address, 0)
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		registration := new(VariableRegistration)
		ABI_register.UnpackVariable(registration, VariableNameRegistration, value)
		if registration.Timestamp > 0 {
			registerList = append(registerList, getAddr(key))
		}
	}
	return registerList
}

// get voters of gid, return map<voter, super node>
func GetVoteMap(db VmDatabase, gid types.Gid) map[types.Address]types.Address {
	iterator := db.NewStorageIterator(gid.Bytes())
	voteMap := make(map[types.Address]types.Address)
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		voterAddr := getAddr(key)
		registerAddr := new(types.Address)
		ABI_vote.UnpackVariable(registerAddr, VariableNameVoteStatus, value)
		voteMap[voterAddr] = *registerAddr
	}
	return voteMap
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
		_, value, ok := iterator.Next()
		if !ok {
			break
		}
		consensusGroupInfo := new(ConsensusGroupInfo)
		ABI_pledge.UnpackVariable(consensusGroupInfo, VariableNameConsensusGroupInfo, value)
		consensusGroupInfoList = append(consensusGroupInfoList, consensusGroupInfo)
	}
	return consensusGroupInfoList
}

// get consensus group info by gid
func GetConsensusGroup(db VmDatabase, gid types.Gid) *ConsensusGroupInfo {
	data := db.GetStorage(&AddressConsensusGroup, types.DataHash(gid.Bytes()).Bytes())
	if len(data) > 0 {
		consensusGroupInfo := new(ConsensusGroupInfo)
		ABI_pledge.UnpackVariable(consensusGroupInfo, VariableNameConsensusGroupInfo, data)
		return consensusGroupInfo
	}
	return nil
}
