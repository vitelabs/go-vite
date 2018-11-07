package chain

import (
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"math/big"
	"strconv"
	"time"

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
var genesisTimestamp = time.Unix(1540200147, 0)

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
		"vite_b1c00ae7dfd5b935550a6e2507da38886abad2351ae78d4d9a",
		"vite_8431a5acb599da19529c3e8bd099087a9d550fb29039dada28",
		"vite_891907ccc07338b84789ce66b4555b03d7e4bb67ce270ba411",
		"vite_9bb3e24453ab22c51d4cb34e1a11149d5a515b7cab1cc50cf9",
		"vite_18e605c19ea97381639921d240ae5222caec8dc3c1a7352033",
		"vite_fbfced2e2678906d4ecf181b8bdfedca8fe75c29388d189a14",
		"vite_0325da1c8782b5ff102ee2cf77d727066d16c98347e703e622",
		"vite_5660160a39215c8bf078dcba134d912b46d58333930991ed47",
		"vite_c840786dae8f182cc2045d6754becf3a2ca63872841ee1d624",
		"vite_f3ebd4ca3302e7ed0ca30b704b918364af81ec626ee1f66726",
		"vite_f2e8ebbf034850cf73a4692687eb5e1df84faad964969f040a",
		"vite_f4d18f7844c31a73ae3e09acaf6524cf9afb320726cad591a7",
		"vite_15acba7c5848c4948b9489255d126ddc434cf7ad0fbbe1c351",
		"vite_13fc91ca355265fcdb52558a7170c2850c51fa4a83bdeb07f4",
		"vite_481fa2e37a577fab051c2c31f82d7d0ea1404dda1fdf8454b7",
		"vite_f89582443e48ca314234d0f8417032f64634f4b72331663b06",
		"vite_86484cad2df02abc763f980f7cbf8ab95995f2dab43b9f57c2",
		"vite_f25da49d7efcc0fa270e456b7906175b63ec55c7a104759a98",
		"vite_43effdd453df7f53e5eca2a15345b122caf2b528f4b1e4f9b2",
		"vite_24a2fad703b14791380afbe8c62bfd8808e2436b513b1b597f",
		"vite_4ba2ffe98b0fabe22057dd55b24c091d7cc0448b38a88e9b62",
		"vite_181600228e5c428565f9b429ec23315161f9b0f2b03561dfc9",
		"vite_9fd1bfbbc0fa526809331185a0a49f161ee35156494652bb0d",
		"vite_7c41774d944a5bf23715c25b47d581c410162d5fa872548ab1",
		"vite_6af8f3c871aa4cd9b1c1697cfb70387bd28d445e4c77ca3a25",
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
