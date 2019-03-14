package chain

import (
	"math/big"
	"strconv"
	"time"

	"github.com/vitelabs/go-vite/vm/contracts/abi"

	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
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

func initGenesis(config *config.Genesis) {
	GenesisSnapshotBlock = NewGenesisSnapshotBlock()

	GenesisMintageBlock, GenesisMintageBlockVC = NewGenesisMintageBlock(config)

	GenesisMintageSendBlock, GenesisMintageSendBlockVC = NewGenesisMintageSendBlock(config)

	GenesisConsensusGroupBlock, GenesisConsensusGroupBlockVC = NewGenesisConsensusGroupBlock(config)

	GenesisRegisterBlock, GenesisRegisterBlockVC = NewGenesisRegisterBlock(config)

	SecondSnapshotBlock = NewSecondSnapshotBlock()
}

var genesisTimestamp = time.Unix(1541650394, 0)

func (c *chain) NewGenesisSnapshotBlock() ledger.SnapshotBlock {
	return NewGenesisSnapshotBlock()
}

func NewGenesisSnapshotBlock() ledger.SnapshotBlock {
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
func (c *chain) NewSecondSnapshotBlock() ledger.SnapshotBlock {
	return NewSecondSnapshotBlock()
}

func NewSecondSnapshotBlock() ledger.SnapshotBlock {
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

func (c *chain) NewGenesisMintageBlock() (ledger.AccountBlock, vmctxt_interface.VmDatabase) {
	return NewGenesisMintageBlock(c.globalCfg.Genesis)
}

func NewGenesisMintageBlock(config *config.Genesis) (ledger.AccountBlock, vmctxt_interface.VmDatabase) {
	timestamp := genesisTimestamp.Add(time.Second * 10)
	block := ledger.AccountBlock{
		BlockType:      ledger.BlockTypeReceive,
		Height:         1,
		AccountAddress: types.AddressMintage,
		Amount:         big.NewInt(0),
		Fee:            big.NewInt(0),

		Timestamp:    &timestamp,
		SnapshotHash: GenesisSnapshotBlock.Hash,
	}

	vmContext := vm_context.NewEmptyVmContextByTrie(trie.NewTrie(nil, nil, nil))
	tokenName := "Vite Token"
	tokenSymbol := "VITE"
	decimals := uint8(18)
	mintageData, _ := abi.ABIMintage.PackVariable(abi.VariableNameMintage, tokenName, tokenSymbol, totalSupply, decimals, config.GenesisAccountAddress, big.NewInt(0), uint64(0))

	vmContext.SetStorage(abi.GetMintageKey(ledger.ViteTokenId), mintageData)

	block.StateHash = *vmContext.GetStorageHash()
	block.Hash = block.ComputeHash()

	return block, vmContext
}

func (c *chain) NewGenesisMintageSendBlock() (ledger.AccountBlock, vmctxt_interface.VmDatabase) {
	return NewGenesisMintageSendBlock(c.globalCfg.Genesis)
}

func NewGenesisMintageSendBlock(config *config.Genesis) (ledger.AccountBlock, vmctxt_interface.VmDatabase) {
	timestamp := genesisTimestamp.Add(time.Second * 12)
	block := ledger.AccountBlock{
		BlockType:      ledger.BlockTypeSendReward,
		PrevHash:       GenesisMintageBlock.Hash,
		Height:         2,
		AccountAddress: types.AddressMintage,
		ToAddress:      config.GenesisAccountAddress,
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

func getConsensusGroupData(consensusGroupConfig *config.ConsensusGroupInfo) ([]byte, error) {

	conditionRegisterData, err := abi.ABIConsensusGroup.PackVariable(abi.VariableNameConditionRegisterOfPledge,
		consensusGroupConfig.RegisterConditionParam.PledgeAmount,
		consensusGroupConfig.RegisterConditionParam.PledgeToken,
		consensusGroupConfig.RegisterConditionParam.PledgeHeight)

	if err != nil {
		return nil, err
	}

	voteConditionData := []byte{}

	if consensusGroupConfig.VoteConditionId > 1 {
		voteConditionData, err = abi.ABIConsensusGroup.PackVariable(abi.VariableNameConditionVoteOfKeepToken,
			consensusGroupConfig.VoteConditionParam.Amount,
			consensusGroupConfig.VoteConditionParam.TokenId)
		if err != nil {
			return nil, err
		}
	}

	return abi.ABIConsensusGroup.PackVariable(abi.VariableNameConsensusGroupInfo,
		consensusGroupConfig.NodeCount,
		consensusGroupConfig.Interval,
		consensusGroupConfig.PerCount,
		consensusGroupConfig.RandCount,
		consensusGroupConfig.RandRank,
		consensusGroupConfig.CountingTokenId,
		consensusGroupConfig.RegisterConditionId,
		conditionRegisterData,
		consensusGroupConfig.VoteConditionId,
		voteConditionData,
		consensusGroupConfig.Owner,
		consensusGroupConfig.PledgeAmount,
		consensusGroupConfig.WithdrawHeight)
}

func (c *chain) NewGenesisConsensusGroupBlock() (ledger.AccountBlock, vmctxt_interface.VmDatabase) {
	return NewGenesisConsensusGroupBlock(c.globalCfg.Genesis)
}

func NewGenesisConsensusGroupBlock(config *config.Genesis) (ledger.AccountBlock, vmctxt_interface.VmDatabase) {
	timestamp := genesisTimestamp.Add(time.Second * 10)

	block := ledger.AccountBlock{
		BlockType:      ledger.BlockTypeReceive,
		Height:         1,
		AccountAddress: types.AddressConsensusGroup,
		Amount:         big.NewInt(0),
		Fee:            big.NewInt(0),

		SnapshotHash: GenesisSnapshotBlock.Hash,
		Timestamp:    &timestamp,
	}

	snapshotConsensusGroupData, err := getConsensusGroupData(config.SnapshotConsensusGroup)
	if err != nil {
		log15.Crit("Init snapshot consensus group information failed, error is "+err.Error(), "module", "genesis")
	}
	commonConsensusGroupData, err := getConsensusGroupData(config.CommonConsensusGroup)
	if err != nil {
		log15.Crit("Init common consensus group information failed, error is "+err.Error(), "module", "genesis")
	}

	vmContext := vm_context.NewEmptyVmContextByTrie(trie.NewTrie(nil, nil, nil))
	vmContext.SetStorage(abi.GetConsensusGroupKey(types.SNAPSHOT_GID), snapshotConsensusGroupData)
	vmContext.SetStorage(abi.GetConsensusGroupKey(types.DELEGATE_GID), commonConsensusGroupData)

	block.StateHash = *vmContext.GetStorageHash()
	block.Hash = block.ComputeHash()

	return block, vmContext
}

func (c *chain) NewGenesisRegisterBlock() (ledger.AccountBlock, vmctxt_interface.VmDatabase) {
	return NewGenesisRegisterBlock(c.globalCfg.Genesis)
}

func NewGenesisRegisterBlock(config *config.Genesis) (ledger.AccountBlock, vmctxt_interface.VmDatabase) {
	timestamp := genesisTimestamp.Add(time.Second * 10)

	block := ledger.AccountBlock{
		BlockType:      ledger.BlockTypeReceive,
		Height:         1,
		AccountAddress: types.AddressConsensusGroup,
		Amount:         big.NewInt(0),
		Fee:            big.NewInt(0),

		SnapshotHash: GenesisSnapshotBlock.Hash,
		Timestamp:    &timestamp,
	}

	vmContext := vm_context.NewEmptyVmContextByTrie(trie.NewTrie(nil, nil, nil))
	for index, addr := range config.BlockProducers {
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
