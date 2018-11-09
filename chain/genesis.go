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
		"vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87",
		"vite_14edbc9214bd1e5f6082438f707d10bf43463a6d599a4f2d08",
		"vite_1630f8c0cf5eda3ce64bd49a0523b826f67b19a33bc2a5dcfb",
		"vite_1b1dfa00323aea69465366d839703547fec5359d6c795c8cef",
		"vite_27a258dd1ed0ce0de3f4abd019adacd1b4b163b879389d3eca",
		"vite_31a02e4f4b536e2d6d9bde23910cdffe72d3369ef6fe9b9239",
		"vite_383fedcbd5e3f52196a4e8a1392ed3ddc4d4360e4da9b8494e",
		"vite_41ba695ff63caafd5460dcf914387e95ca3a900f708ac91f06",
		"vite_545c8e4c74e7bb6911165e34cbfb83bc513bde3623b342d988",
		"vite_5a1b5ece654138d035bdd9873c1892fb5817548aac2072992e",
		"vite_70cfd586185e552635d11f398232344f97fc524fa15952006d",
		"vite_76df2a0560694933d764497e1b9b11f9ffa1524b170f55dda0",
		"vite_7b76ca2433c7ddb5a5fa315ca861e861d432b8b05232526767",
		"vite_7caaee1d51abad4047a58f629f3e8e591247250dad8525998a",
		"vite_826a1ab4c85062b239879544dc6b67e3b5ce32d0a1eba21461",
		"vite_89007189ad81c6ee5cdcdc2600a0f0b6846e0a1aa9a58e5410",
		"vite_9abcb7324b8d9029e4f9effe76f7336bfd28ed33cb5b877c8d",
		"vite_af60cf485b6cc2280a12faac6beccfef149597ea518696dcf3",
		"vite_c1090802f735dfc279a6c24aacff0e3e4c727934e547c24e5e",
		"vite_c10ae7a14649800b85a7eaaa8bd98c99388712412b41908cc0",
		"vite_d45ac37f6fcdb1c362a33abae4a7d324a028aa49aeea7e01cb",
		"vite_d8974670af8e1f3c4378d01d457be640c58644bc0fa87e3c30",
		"vite_e289d98f33c3ef5f1b41048c2cb8b389142f033d1df9383818",
		"vite_f53dcf7d40b582cd4b806d2579c6dd7b0b131b96c2b2ab5218",
		"vite_fac06662d84a7bea269265e78ea2d9151921ba2fae97595608",
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
