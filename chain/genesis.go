package chain

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"strconv"
	"time"
)

var GenesisSnapshotBlock ledger.SnapshotBlock

var GenesisMintageBlock ledger.AccountBlock
var GenesisMintageBlockVC vmctxt_interface.VmDatabase

var GenesisMintageSendBlock ledger.AccountBlock
var GenesisMintageSendBlockVC vmctxt_interface.VmDatabase

var GenesisConsensusGroupBlock ledger.AccountBlock
var GenesisConsensusGroupBlockVC vmctxt_interface.VmDatabase

var GenesisRegisterBlock ledger.AccountBlock
var GenesisRegisterBlockVC vmctxt_interface.VmDatabase

func init() {
	GenesisMintageBlock, GenesisMintageBlockVC = genesisMintageBlock()

	GenesisMintageSendBlock, GenesisMintageSendBlockVC = genesisMintageSendBlock()

	GenesisConsensusGroupBlock, GenesisConsensusGroupBlockVC = genesisConsensusGroupBlock()

	GenesisRegisterBlock, GenesisRegisterBlockVC = genesisRegisterBlock()

	GenesisSnapshotBlock = genesisSnapshotBlock()
}

var genesisTimestamp = time.Unix(1537361101, 0)

func genesisSnapshotBlock() ledger.SnapshotBlock {
	genesisSnapshotBlock := ledger.SnapshotBlock{
		Height:    1,
		Timestamp: &genesisTimestamp,
		PublicKey: ledger.GenesisPublicKey,
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

	genesisSnapshotBlock.Signature, _ = hex.DecodeString("42aa62748c3655e4a911b4d68fd8646d66ec8e2e5a71cd94df3d0f776f7a58d83ff70afd5ab4da158955a3c550bd9943d67d0091c5fb00975696c0703d535608")

	return genesisSnapshotBlock
}

var totalSupply = new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e9))

func genesisMintageBlock() (ledger.AccountBlock, vmctxt_interface.VmDatabase) {
	timestamp := genesisTimestamp.Add(time.Second * 10)
	block := ledger.AccountBlock{
		BlockType:      ledger.BlockTypeReceive,
		Height:         1,
		AccountAddress: contracts.AddressMintage,
		PublicKey:      ledger.GenesisPublicKey,
		Amount:         big.NewInt(0),
		Fee:            big.NewInt(0),

		Timestamp: &timestamp,
	}

	vmContext := vm_context.NewEmptyVmContextByTrie(nil)
	tokenName := "Vite Token"
	tokenSymbol := "VITE"
	decimals := uint8(18)
	mintageData, _ := contracts.ABIMintage.PackVariable(contracts.VariableNameMintage, tokenName, tokenSymbol, totalSupply, decimals, ledger.GenesisAccountAddress, big.NewInt(0), int64(0))

	vmContext.SetStorage(contracts.GetMintageKey(ledger.ViteTokenId), mintageData)

	block.StateHash = *vmContext.GetStorageHash()
	block.Hash = block.ComputeHash()
	block.Signature, _ = hex.DecodeString("1fa7f85de753e1741ba7b92cbc01c49f7dd54f0d3815e7f368d95d8c511499dde1b628944f1f85c0670ecba54be08c7374c165cae540fee4ac517cc825469003")

	return block, vmContext
}

func genesisMintageSendBlock() (ledger.AccountBlock, vmctxt_interface.VmDatabase) {
	timestamp := genesisTimestamp.Add(time.Second * 12)
	block := ledger.AccountBlock{
		BlockType:      ledger.BlockTypeSendReward,
		PrevHash:       GenesisMintageBlock.Hash,
		Height:         2,
		AccountAddress: contracts.AddressMintage,
		ToAddress:      ledger.GenesisAccountAddress,
		PublicKey:      ledger.GenesisPublicKey,
		Amount:         totalSupply,
		TokenId:        ledger.ViteTokenId,
		Fee:            big.NewInt(0),
		StateHash:      GenesisMintageBlock.StateHash,
		Timestamp:      &timestamp,
	}
	block.Hash = block.ComputeHash()
	block.Signature, _ = hex.DecodeString("0db9eea19460d90fce2d6f307c132061f3a68d89bdd9ac2768383dbd9774f15784073795be442fa7335a61532887ff3f6ee93a59d694b95660ae04383eb9f40d")

	return block, GenesisMintageBlockVC.CopyAndFreeze()
}

func genesisConsensusGroupBlock() (ledger.AccountBlock, vmctxt_interface.VmDatabase) {
	timestamp := genesisTimestamp.Add(time.Second * 10)

	block := ledger.AccountBlock{
		BlockType:      ledger.BlockTypeReceive,
		Height:         1,
		AccountAddress: contracts.AddressConsensusGroup,
		PublicKey:      ledger.GenesisPublicKey, //todo
		Amount:         big.NewInt(0),
		Fee:            big.NewInt(0),

		Timestamp: &timestamp,
	}

	conditionRegisterData, _ := contracts.ABIConsensusGroup.PackVariable(contracts.VariableNameConditionRegisterOfPledge, new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18)), ledger.ViteTokenId, int64(3600*24*90))

	snapshotConsensusGroupData, _ := contracts.ABIConsensusGroup.PackVariable(contracts.VariableNameConsensusGroupInfo,
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
		ledger.GenesisAccountAddress,
		big.NewInt(0),
		int64(1))

	commonConsensusGroupData, _ := contracts.ABIConsensusGroup.PackVariable(contracts.VariableNameConsensusGroupInfo,
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
		ledger.GenesisAccountAddress,
		big.NewInt(0),
		int64(1))

	vmContext := vm_context.NewEmptyVmContextByTrie(nil)
	vmContext.SetStorage(contracts.GetConsensusGroupKey(types.SNAPSHOT_GID), snapshotConsensusGroupData)
	vmContext.SetStorage(contracts.GetConsensusGroupKey(types.DELEGATE_GID), commonConsensusGroupData)

	block.StateHash = *vmContext.GetStorageHash()
	block.Hash = block.ComputeHash()
	block.Signature, _ = hex.DecodeString("46aa9174c006665da1461961cfdd4dda7133c7682e89672296ef21e76ebb56435df65f6e10b3af3012a53e22e62f469f80a156f659a4d32fc6e7e40c1e353701")

	return block, vmContext
}

func genesisRegisterBlock() (ledger.AccountBlock, vmctxt_interface.VmDatabase) {
	timestamp := genesisTimestamp.Add(time.Second * 10)

	block := ledger.AccountBlock{
		BlockType:      ledger.BlockTypeReceive,
		Height:         1,
		AccountAddress: contracts.AddressRegister,
		PublicKey:      ledger.GenesisPublicKey, //todo
		Amount:         big.NewInt(0),
		Fee:            big.NewInt(0),

		Timestamp: &timestamp,
	}

	addrStrList := []string{
		"vite_39f1ede9ab4979b8a77167bfade02a3b4df0c413ad048cb999",
		"vite_d7e73d4c7d07746e5015f39e7e037fe4d3577585e0a60ea3b4",
		"vite_e9b7307aaf51818993bb2675fd26a600bc7ab6d0f52bc5c2c1",
		"vite_73d6bf04045038a724a7b91eedeecfffabf04e6d9209994dd4",
		"vite_a60e507124c2059ccb61039870dac3b5219aca014abc3807d0",
		"vite_93a9bd77fb0ae68d4f23219afe08f631d5a6f30f2f8bf62341",
		"vite_aad4536fd06e37461ae1baabcf2a7bd958a77b1a739bb05822",
		"vite_42f326d36cad0fb772552943e6805c3bf0dfd7e837cfd5dc15",
		"vite_9cfaab0b94b3c538db63f9219cf3701173e8b5ef0b15e046a9",
		"vite_49696c180e80acc668a567e30187797e5b18e0c7abf33014e2",
		"vite_918715bae54a3f221c1bcb3885467326a6d10c234fb6ca93a1",
		"vite_da36f3623e2a60817573ff6b407f67c5c16165c5a0356c7b0e",
		"vite_5d4ddbdb4b541210fc9d7c2cb28c46714ab0c64b6c8564e60a",
		"vite_a1c1342c0651a7743e687890f749e1f08d7e1f447ad8ebf896",
		"vite_9acca69467d586d80fe44f7fd39c93fa5c2b5e0ac267f9deb5",
		"vite_d4c974b6d8ec30f32add577d417e62b34dd94f122d5ba49000",
		"vite_7f85cc5b6a4d2f52955b811aad9db3e8b201adb92942a146be",
		"vite_3d1d81b4a578885774bbfe471d1111be1da636f18cd13cbace",
		"vite_eec4df5cd039c2590665251e524715f4e67e0034af556bb793",
		"vite_7eb26bd846c416db979800e88f4ba2616a1093645b1b77726a",
		"vite_5eff9251b046b5c4d8659c80e8db85aa5a131b51894f8a35ea",
		"vite_fc551abb96ed191b081c61c31e08ada8e02cd4d15261a5213e",
		"vite_72857dc260843374244e859dba7cba5de8c0db919ca2a444a8",
		"vite_f13e8ca8c1c05aeec545c35a1e573b11563fac07fd634dca16",
		"vite_059b6ec803981b75c2e36e8e6f9acffb142c7a245cd025989e",
	}
	var addrList []types.Address

	for _, addrStr := range addrStrList {
		addr, _ := types.HexToAddress(addrStr)
		addrList = append(addrList, addr)
	}

	vmContext := vm_context.NewEmptyVmContextByTrie(nil)
	for index, addr := range addrList {
		nodeName := "s" + strconv.Itoa(index+1)
		registerData, _ := contracts.ABIRegister.PackVariable(contracts.VariableNameRegistration, nodeName, addr, addr, addr, helper.Big0, int64(1), uint64(1), uint64(0))
		vmContext.SetStorage(contracts.GetRegisterKey(nodeName, types.SNAPSHOT_GID), registerData)
	}

	block.StateHash = *vmContext.GetStorageHash()
	block.Hash = block.ComputeHash()
	block.Signature, _ = hex.DecodeString("169c1c1dc64fadc3067c173ecda8972a87b912871a9d8a0318f35a69939fc1ad1235f8cbe38ad5e0770a4a82f3b5f81c3aab7516d8dc1c6525b23f0ee7fc7505")

	return block, vmContext
}
