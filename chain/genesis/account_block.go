package chain_genesis

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
	"strconv"
)

func NewGenesisAccountBlocks() []*vm_db.VmAccountBlock {
	list := make([]*vm_db.VmAccountBlock, 3)
	list[0] = NewGenesisConsensusGroupBlock()
	list[1] = NewGenesisMintageBlock()
	list[2] = NewGenesisAddressBlock()
	return list
}

var totalSupply = new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e9))
var genesisAccountAddress, _ = types.HexToAddress("vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a")

func NewGenesisMintageBlock() *vm_db.VmAccountBlock {
	block := ledger.AccountBlock{
		BlockType:      ledger.BlockTypeGenesisReceive,
		Height:         1,
		AccountAddress: types.AddressMintage,
		Amount:         big.NewInt(0),
		Fee:            big.NewInt(0),
	}

	vmdb := vm_db.NewEmptyVmDB(&types.AddressMintage)
	tokenName := "Vite Token"
	tokenSymbol := "VITE"
	decimals := uint8(18)
	mintageData, _ := abi.ABIMintage.PackVariable(abi.VariableNameTokenInfo,
		tokenName, tokenSymbol, totalSupply, decimals, genesisAccountAddress, big.NewInt(0), uint64(0), genesisAccountAddress, true, helper.Tt256m1, false)

	vmdb.SetValue(abi.GetMintageKey(ledger.ViteTokenId), mintageData)
	block.Hash = block.ComputeHash()

	return &vm_db.VmAccountBlock{&block, vmdb}
}

var (
	blockProducers = []string{
		"vite_14edbc9214bd1e5f6082438f707d10bf43463a6d599a4f2d08",
		"vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87",
		"vite_27a258dd1ed0ce0de3f4abd019adacd1b4b163b879389d3eca",
		"vite_1b1dfa00323aea69465366d839703547fec5359d6c795c8cef",
		"vite_1630f8c0cf5eda3ce64bd49a0523b826f67b19a33bc2a5dcfb",
		"vite_383fedcbd5e3f52196a4e8a1392ed3ddc4d4360e4da9b8494e",
		"vite_31a02e4f4b536e2d6d9bde23910cdffe72d3369ef6fe9b9239",
		"vite_70cfd586185e552635d11f398232344f97fc524fa15952006d",
		"vite_41ba695ff63caafd5460dcf914387e95ca3a900f708ac91f06",
		"vite_545c8e4c74e7bb6911165e34cbfb83bc513bde3623b342d988",
		"vite_5a1b5ece654138d035bdd9873c1892fb5817548aac2072992e",
	}
)

func NewGenesisConsensusGroupBlock() *vm_db.VmAccountBlock {
	block := ledger.AccountBlock{
		BlockType:      ledger.BlockTypeGenesisReceive,
		Height:         1,
		AccountAddress: types.AddressConsensusGroup,
		Amount:         big.NewInt(0),
		Fee:            big.NewInt(0),
	}

	vmdb := vm_db.NewEmptyVmDB(&types.AddressConsensusGroup)

	conditionRegisterData, err := abi.ABIConsensusGroup.PackVariable(abi.VariableNameConditionRegisterOfPledge, new(big.Int).Mul(big.NewInt(1e5), util.AttovPerVite), ledger.ViteTokenId, uint64(3600*24*90))
	if err != nil {
		panic(err)
	}
	snapshotConsensusGroupData, err := abi.ABIConsensusGroup.PackVariable(abi.VariableNameConsensusGroupInfo,
		uint8(25),
		int64(1),
		int64(3),
		uint8(2),
		uint8(50),
		ledger.ViteTokenId,
		uint8(1),
		conditionRegisterData,
		uint8(1),
		[]byte{},
		genesisAccountAddress,
		big.NewInt(0),
		uint64(1))
	if err != nil {
		panic(err)
	}
	vmdb.SetValue(abi.GetConsensusGroupKey(types.SNAPSHOT_GID), snapshotConsensusGroupData)
	commonConsensusGroupData, err := abi.ABIConsensusGroup.PackVariable(abi.VariableNameConsensusGroupInfo,
		uint8(25),
		int64(3),
		int64(1),
		uint8(2),
		uint8(50),
		ledger.ViteTokenId,
		uint8(1),
		conditionRegisterData,
		uint8(1),
		[]byte{},
		genesisAccountAddress,
		big.NewInt(0),
		uint64(1))
	vmdb.SetValue(abi.GetConsensusGroupKey(types.DELEGATE_GID), commonConsensusGroupData)
	for index, addrStr := range blockProducers {
		nodeName := "s" + strconv.Itoa(index+1)
		addr, _ := types.HexToAddress(addrStr)
		registerData, _ := abi.ABIConsensusGroup.PackVariable(abi.VariableNameRegistration, nodeName, addr, addr, helper.Big0, uint64(1), int64(1), int64(0), []types.Address{addr})
		vmdb.SetValue(abi.GetRegisterKey(nodeName, types.SNAPSHOT_GID), registerData)
		hisNameData, _ := abi.ABIConsensusGroup.PackVariable(abi.VariableNameHisName, nodeName)
		vmdb.SetValue(abi.GetHisNameKey(addr, types.SNAPSHOT_GID), hisNameData)
	}

	block.Hash = block.ComputeHash()
	return &vm_db.VmAccountBlock{&block, vmdb}
}

func NewGenesisAddressBlock() *vm_db.VmAccountBlock {
	block := ledger.AccountBlock{
		BlockType:      ledger.BlockTypeGenesisReceive,
		Height:         1,
		AccountAddress: genesisAccountAddress,
		Amount:         big.NewInt(0),
		Fee:            big.NewInt(0),
	}

	vmdb := vm_db.NewEmptyVmDB(&genesisAccountAddress)
	vmdb.SetBalance(&ledger.ViteTokenId, totalSupply)

	block.Hash = block.ComputeHash()
	return &vm_db.VmAccountBlock{&block, vmdb}
}
