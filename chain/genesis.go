package chain

import (
	"math/big"
	"strconv"
	"time"

	"github.com/vitelabs/go-vite/vm/contracts/abi"

	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
)

var GenesisSnapshotBlock ledger.SnapshotBlock
var SecondSnapshotBlock ledger.SnapshotBlock

var GenesisMintageBlock ledger.AccountBlock
var GenesisMintageBlockVC vmctxt_interface.VmDatabase

var GenesisMintageSendBlock ledger.AccountBlock
var GenesisMintageSendBlockVC vmctxt_interface.VmDatabase

var GenesisConsensusGroupBlock ledger.AccountBlock
var GenesisConsensusGroupBlockVC vmctxt_interface.VmDatabase

var GenesisRegisterBlock ledger.AccountBlock
var GenesisRegisterBlockVC vmctxt_interface.VmDatabase

func init() {
	GenesisSnapshotBlock = genesisSnapshotBlock()

	GenesisMintageBlock, GenesisMintageBlockVC = genesisMintageBlock()

	GenesisMintageSendBlock, GenesisMintageSendBlockVC = genesisMintageSendBlock()

	GenesisConsensusGroupBlock, GenesisConsensusGroupBlockVC = genesisConsensusGroupBlock()

	GenesisRegisterBlock, GenesisRegisterBlockVC = genesisRegisterBlock()

	SecondSnapshotBlock = secondSnapshotBlock()
}

var genesisTrieNodePool = trie.NewTrieNodePool()
var genesisTimestamp = time.Unix(1541650394, 0)

func genesisSnapshotBlock() ledger.SnapshotBlock {
	genesisSnapshotBlock := ledger.SnapshotBlock{
		Height:    1,
		Timestamp: &genesisTimestamp,
	}
	stateTrie := trie.NewTrie(nil, nil, nil)
	stateTrie.SetValue([]byte("vite"), []byte("create something cool"))

	genesisSnapshotBlock.StateTrie = stateTrie
	genesisSnapshotBlock.StateHash = *stateTrie.Hash()

	genesisSnapshotBlock.Hash = genesisSnapshotBlock.ComputeHash()

	return genesisSnapshotBlock
}

func secondSnapshotBlock() ledger.SnapshotBlock {
	timestamp := genesisTimestamp.Add(time.Second * 15)

	genesisSnapshotBlock := ledger.SnapshotBlock{
		Height:    GenesisSnapshotBlock.Height + 1,
		Timestamp: &timestamp,
		PrevHash:  GenesisSnapshotBlock.Hash,
	}

	snapshotContent := ledger.SnapshotContent{
		GenesisMintageSendBlock.AccountAddress: &ledger.HashHeight{
			Hash:   GenesisMintageSendBlock.Hash,
			Height: GenesisMintageSendBlock.Height,
		},
		GenesisConsensusGroupBlock.AccountAddress: &ledger.HashHeight{
			Hash:   GenesisConsensusGroupBlock.Hash,
			Height: GenesisConsensusGroupBlock.Height,
		},
		GenesisRegisterBlock.AccountAddress: &ledger.HashHeight{
			Hash:   GenesisRegisterBlock.Hash,
			Height: GenesisRegisterBlock.Height,
		},
	}

	genesisSnapshotBlock.SnapshotContent = snapshotContent
	stateTrie := trie.NewTrie(nil, nil, nil)
	stateTrie.SetValue(GenesisMintageSendBlock.AccountAddress.Bytes(), GenesisMintageSendBlock.StateHash.Bytes())
	stateTrie.SetValue(GenesisConsensusGroupBlock.AccountAddress.Bytes(), GenesisConsensusGroupBlock.StateHash.Bytes())
	stateTrie.SetValue(GenesisRegisterBlock.AccountAddress.Bytes(), GenesisRegisterBlock.StateHash.Bytes())

	genesisSnapshotBlock.StateHash = *stateTrie.Hash()
	genesisSnapshotBlock.StateTrie = stateTrie
	genesisSnapshotBlock.Hash = genesisSnapshotBlock.ComputeHash()

	return genesisSnapshotBlock
}

var totalSupply = new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e9))

func genesisMintageBlock() (ledger.AccountBlock, vmctxt_interface.VmDatabase) {
	timestamp := genesisTimestamp.Add(time.Second * 10)
	block := ledger.AccountBlock{
		BlockType:      ledger.BlockTypeReceive,
		Height:         1,
		AccountAddress: abi.AddressMintage,
		Amount:         big.NewInt(0),
		Fee:            big.NewInt(0),

		Timestamp:    &timestamp,
		SnapshotHash: GenesisSnapshotBlock.Hash,
	}

	vmContext := vm_context.NewEmptyVmContextByTrie(trie.NewTrie(nil, nil, genesisTrieNodePool))
	tokenName := "Vite Token"
	tokenSymbol := "VITE"
	decimals := uint8(18)
	mintageData, _ := abi.ABIMintage.PackVariable(abi.VariableNameMintage, tokenName, tokenSymbol, totalSupply, decimals, ledger.GenesisAccountAddress, big.NewInt(0), uint64(0))

	vmContext.SetStorage(abi.GetMintageKey(ledger.ViteTokenId), mintageData)

	block.StateHash = *vmContext.GetStorageHash()
	block.Hash = block.ComputeHash()

	return block, vmContext
}

func genesisMintageSendBlock() (ledger.AccountBlock, vmctxt_interface.VmDatabase) {
	timestamp := genesisTimestamp.Add(time.Second * 12)
	block := ledger.AccountBlock{
		BlockType:      ledger.BlockTypeSendReward,
		PrevHash:       GenesisMintageBlock.Hash,
		Height:         2,
		AccountAddress: abi.AddressMintage,
		ToAddress:      ledger.GenesisAccountAddress,
		Amount:         totalSupply,
		TokenId:        ledger.ViteTokenId,
		Fee:            big.NewInt(0),
		StateHash:      GenesisMintageBlock.StateHash,
		SnapshotHash:   GenesisSnapshotBlock.Hash,
		Timestamp:      &timestamp,
	}
	block.Hash = block.ComputeHash()

	return block, GenesisMintageBlockVC.CopyAndFreeze()
}

func genesisConsensusGroupBlock() (ledger.AccountBlock, vmctxt_interface.VmDatabase) {
	timestamp := genesisTimestamp.Add(time.Second * 10)

	block := ledger.AccountBlock{
		BlockType:      ledger.BlockTypeReceive,
		Height:         1,
		AccountAddress: abi.AddressConsensusGroup,
		Amount:         big.NewInt(0),
		Fee:            big.NewInt(0),

		SnapshotHash: GenesisSnapshotBlock.Hash,
		Timestamp:    &timestamp,
	}

	conditionRegisterData, _ := abi.ABIConsensusGroup.PackVariable(abi.VariableNameConditionRegisterOfPledge, new(big.Int).Mul(big.NewInt(5e5), big.NewInt(1e18)), ledger.ViteTokenId, uint64(3600*24*90))

	snapshotConsensusGroupData, _ := abi.ABIConsensusGroup.PackVariable(abi.VariableNameConsensusGroupInfo,
		uint8(25),
		int64(1),
		int64(3),
		uint8(2),
		uint8(100),
		ledger.ViteTokenId,
		uint8(1),
		conditionRegisterData,
		uint8(1),
		[]byte{},
		ledger.GenesisAccountAddress,
		big.NewInt(0),
		uint64(1))

	commonConsensusGroupData, _ := abi.ABIConsensusGroup.PackVariable(abi.VariableNameConsensusGroupInfo,
		uint8(25),
		int64(3),
		int64(1),
		uint8(2),
		uint8(100),
		ledger.ViteTokenId,
		uint8(1),
		conditionRegisterData,
		uint8(1),
		[]byte{},
		ledger.GenesisAccountAddress,
		big.NewInt(0),
		uint64(1))

	vmContext := vm_context.NewEmptyVmContextByTrie(trie.NewTrie(nil, nil, genesisTrieNodePool))
	vmContext.SetStorage(abi.GetConsensusGroupKey(types.SNAPSHOT_GID), snapshotConsensusGroupData)
	vmContext.SetStorage(abi.GetConsensusGroupKey(types.DELEGATE_GID), commonConsensusGroupData)

	block.StateHash = *vmContext.GetStorageHash()
	block.Hash = block.ComputeHash()

	return block, vmContext
}

func genesisRegisterBlock() (ledger.AccountBlock, vmctxt_interface.VmDatabase) {
	timestamp := genesisTimestamp.Add(time.Second * 10)

	block := ledger.AccountBlock{
		BlockType:      ledger.BlockTypeReceive,
		Height:         1,
		AccountAddress: abi.AddressRegister,
		Amount:         big.NewInt(0),
		Fee:            big.NewInt(0),

		SnapshotHash: GenesisSnapshotBlock.Hash,
		Timestamp:    &timestamp,
	}

	addrStrList := []string{
		"vite_74f62d52eb59f52fcc7c71ccdb0ac2cc8e00f5107f02fb9613",
		"vite_06e139359b64f8f0d297e6ce78ab4e3ac7bd66ed7f16fe0778",
		"vite_05149c03edf3b74bbef4c49c29c1486d49e0ea4150ba64eed4",
		"vite_68bbc369fa0bb7ddbdd70abca83d2c88c574a1aa53c1e3491d",
		"vite_1e7108a1e8730835859e609c9fb97b0540e758c1ffb8248f01",
		"vite_6c1cf3362515c7f779ee4a813ad433b2aad79b3fd412065af3",
		"vite_19771456d1c8a92fc51076345348c36ecfb86ccb22384514b5",
		"vite_eeccec884b67411008a0cf0fdce3ce91b9bc37e9cd4380ec2a",
		"vite_61adec0b781c381d08dc6ce3eb214dc9de84c997e700c7bc8c",
		"vite_54413c0c0fecfc7933fbe0d6d5b9390cbb27332ace44ef8342",
		"vite_4942542759d202be78cab0497631fd2a0e8ca7c6235b75f55a",
		"vite_621d2ec0febdd2d6be38cbda2ead227cd77d38908fb7d57bc6",
		"vite_65a78201544b1d20f4590c8de255b4ed03c9de513904ff1437",
		"vite_f14df2a0f90d001c2f6647770802777c1a8c412614ade18619",
		"vite_0a020b3288da4f53b3cdf4b2e1af71a298be04c470193171eb",
		"vite_87acebd814554e755360b6b0ead41d907d4acfe00435a05242",
		"vite_83f304da8c3e3863ea8a8332e8a80ad9f40c3a4e4c45db6ee5",
		"vite_c3952370e046609808717e1af9352a9474e6587355dc008751",
		"vite_a179f525d1de712bf6feebb451c9dd82606844e9859fdab2fd",
		"vite_3c9c2f88d797ba4c5c1f01b1629823777b06cbd6c8f9b655ee",
		"vite_2b32ba6d22502d6f18be400e27a92a3e24634346764c57b961",
		"vite_e3ad0b085e55f77e9452ee5c39ea3742b3def8e9646c52673a",
		"vite_40dfb74244b6bfaf36f8b0ac7396b0382447ce8d200e1966da",
		"vite_04c149177ad8e6b52786bd6e45efe5f6898495c14a5c1d0ed0",
		"vite_fb8cd0377eac3979db459325f1404da945ad6e0d82d554faad",
	}
	var addrList []types.Address

	for _, addrStr := range addrStrList {
		addr, _ := types.HexToAddress(addrStr)
		addrList = append(addrList, addr)
	}

	vmContext := vm_context.NewEmptyVmContextByTrie(trie.NewTrie(nil, nil, genesisTrieNodePool))
	for index, addr := range addrList {
		nodeName := "s" + strconv.Itoa(index+1)
		registerData, _ := abi.ABIRegister.PackVariable(abi.VariableNameRegistration, nodeName, addr, addr, helper.Big0, uint64(1), uint64(0), uint64(0), []types.Address{addr})
		vmContext.SetStorage(abi.GetRegisterKey(nodeName, types.SNAPSHOT_GID), registerData)
		hisNameData, _ := abi.ABIRegister.PackVariable(abi.VariableNameHisName, nodeName)
		vmContext.SetStorage(abi.GetHisNameKey(addr, types.SNAPSHOT_GID), hisNameData)
	}

	block.StateHash = *vmContext.GetStorageHash()
	block.Hash = block.ComputeHash()

	return block, vmContext
}
