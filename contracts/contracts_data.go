package contracts

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
)

var (
	AddressRegister, _       = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	AddressVote, _           = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2})
	AddressPledge, _         = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3})
	AddressConsensusGroup, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4})
	AddressMintage, _        = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5})
)

type StorageDatabase interface {
	GetStorage(addr *types.Address, key []byte) []byte
	NewStorageIterator(prefix []byte) vmctxt_interface.StorageIterator
}

func GetTokenById(db StorageDatabase, tokenId types.TokenTypeId) *TokenInfo {
	data := db.GetStorage(&AddressMintage, helper.LeftPadBytes(tokenId.Bytes(), types.HashSize))
	if len(data) > 0 {
		tokenInfo := new(TokenInfo)
		ABI_mintage.UnpackVariable(tokenInfo, VariableNameMintage, data)
		return tokenInfo
	}
	return nil
}

func GetTokenMap(db StorageDatabase) map[types.TokenTypeId]*TokenInfo {
	iterator := db.NewStorageIterator(nil)
	tokenInfoMap := make(map[types.TokenTypeId]*TokenInfo)
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		tokenId, _ := types.BytesToTokenTypeId(key[types.HashSize-types.TokenTypeIdSize:])
		tokenInfo := new(TokenInfo)
		ABI_mintage.UnpackVariable(tokenInfo, VariableNameMintage, value)
		tokenInfoMap[tokenId] = tokenInfo
	}
	return tokenInfoMap
}

func GetRegisterList(db StorageDatabase, gid types.Gid) []*Registration {
	iterator := db.NewStorageIterator(gid.Bytes())
	registerList := make([]*Registration, 0)
	for {
		_, value, ok := iterator.Next()
		if !ok {
			break
		}
		registration := new(Registration)
		ABI_register.UnpackVariable(registration, VariableNameRegistration, value)
		if registration.CancelHeight == 0 {
			registerList = append(registerList, registration)
		}
	}
	return registerList
}

func GetVoteList(db StorageDatabase, gid types.Gid) []*VoteInfo {
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

func GetPledgeAmount(db StorageDatabase, beneficial types.Address) *big.Int {
	locHash := types.DataHash(beneficial.Bytes()).Bytes()
	beneficialAmount := new(VariablePledgeBeneficial)
	err := ABI_pledge.UnpackVariable(beneficialAmount, VariableNamePledgeBeneficial, db.GetStorage(&AddressPledge, locHash))
	if err == nil {
		return beneficialAmount.Amount
	}
	return big.NewInt(0)
}

func GetConsensusGroupList(db StorageDatabase) []*ConsensusGroupInfo {
	iterator := db.NewStorageIterator(nil)
	consensusGroupInfoList := make([]*ConsensusGroupInfo, 0)
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		consensusGroupInfo := new(ConsensusGroupInfo)
		ABI_consensusGroup.UnpackVariable(consensusGroupInfo, VariableNameConsensusGroupInfo, value)
		consensusGroupInfo.Gid = GetGidFromConsensusGroupKey(key)
		consensusGroupInfoList = append(consensusGroupInfoList, consensusGroupInfo)
	}
	return consensusGroupInfoList
}

func GetConsensusGroup(db StorageDatabase, gid types.Gid) *ConsensusGroupInfo {
	data := db.GetStorage(&AddressConsensusGroup, GetConsensusGroupKey(gid))
	if len(data) > 0 {
		consensusGroupInfo := new(ConsensusGroupInfo)
		ABI_consensusGroup.UnpackVariable(consensusGroupInfo, VariableNameConsensusGroupInfo, data)
		consensusGroupInfo.Gid = gid
		return consensusGroupInfo
	}
	return nil
}

func GetGidFromCreateContractData(data []byte) types.Gid {
	gid, _ := types.BytesToGid(data[:types.GidSize])
	return gid
}
