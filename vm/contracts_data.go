package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

// get register address of gid
func GetRegisterList(db VmDatabase, gid types.Gid) []types.Address {
	iterator := db.GetDbIteratorByPrefix(gid.Bytes())
	registerList := make([]types.Address, 0)
	for iterator.HasNext() {
		key, value := iterator.Next()
		registration := new(VariableRegistration)
		ABI_register.UnpackVariable(registration, VariableNameRegistration, value)
		if registration.Amount.Sign() > 0 {
			registerList = append(registerList, getAddr(key))
		}
	}
	return registerList
}

// get voters of gid, return map<voter, super node>
func GetVoteMap(db VmDatabase, gid types.Gid) map[types.Address]types.Address {
	iterator := db.GetDbIteratorByPrefix(gid.Bytes())
	voteMap := make(map[types.Address]types.Address)
	for iterator.HasNext() {
		key, value := iterator.Next()
		voterAddr := getAddr(key)
		registerAddr := new(types.Address)
		ABI_vote.UnpackVariable(registerAddr, VariableNameVoteStatus, value)
		voteMap[voterAddr] = *registerAddr
	}
	return voteMap
}

// get beneficial pledge amount
func GetPledgeAmount(db VmDatabase, beneficial types.Address) *big.Int {
	locHash := types.DataHash(beneficial.Bytes())
	beneficialAmount := new(VariablePledgeBeneficial)
	err := ABI_pledge.UnpackVariable(beneficialAmount, VariableNamePledgeBeneficial, db.Storage(AddressPledge, locHash))
	if err != nil {
		return beneficialAmount.Amount
	}
	return big.NewInt(0)
}

// get all consensus group info
func GetConsensusGroupList(db VmDatabase) []*VariableConsensusGroupInfo {
	iterator := db.GetDbIteratorByPrefix(nil)
	consensusGroupInfoList := make([]*VariableConsensusGroupInfo, 0)
	for iterator.HasNext() {
		_, value := iterator.Next()
		consensusGroupInfo := new(VariableConsensusGroupInfo)
		ABI_pledge.UnpackVariable(consensusGroupInfo, VariableNameConsensusGroupInfo, value)
		consensusGroupInfoList = append(consensusGroupInfoList, consensusGroupInfo)
	}
	return consensusGroupInfoList
}
