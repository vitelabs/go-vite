package config_gen

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"

	"github.com/vitelabs/go-vite/config"

	"github.com/ethereum/go-ethereum/log"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
)

func MakeGenesisConfig(genesisFile string) *config.Genesis {
	var genesisConfig *config.Genesis

	if len(genesisFile) > 0 {
		file, err := os.Open(genesisFile)
		if err != nil {
			log.Crit(fmt.Sprintf("Failed to read genesis file: %v", err), "method", "readGenesis")
		}
		defer file.Close()

		genesisConfig = new(config.Genesis)
		if err := json.NewDecoder(file).Decode(genesisConfig); err != nil {
			log.Crit(fmt.Sprintf("invalid genesis file: %v", err), "method", "readGenesis")
		}
		if genesisConfig == nil || genesisConfig.GenesisAccountAddress == nil || len(genesisConfig.ContractStorageMap) == 0 || len(genesisConfig.AccountBalanceMap) == 0 {
			log.Crit(fmt.Sprintf("invalid genesis file, genesis account info is not complete"), "method", "readGenesis")
		}
	} else {
		genesisConfig = makeGenesisAccountConfig()
	}

	// set fork points
	genesisConfig.ForkPoints = makeForkPointsConfig(genesisConfig)

	return genesisConfig
}

func makeForkPointsConfig(genesisConfig *config.Genesis) *config.ForkPoints {
	forkPoints := &config.ForkPoints{}

	if genesisConfig != nil && genesisConfig.ForkPoints != nil {
		forkPoints = genesisConfig.ForkPoints
	}

	return forkPoints
}

func makeGenesisAccountConfig() *config.Genesis {
	defaultGenesisAccountAddress, _ := types.HexToAddress("vite_60e292f0ac471c73d914aeff10bb25925e13b2a9fddb6e6122")

	contractStorageMap := make(map[string]map[string]string)
	contractStorageMap[types.AddressConsensusGroup.String()] = make(map[string]string)
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
		uint16(1),
		uint8(0),
		ledger.ViteTokenId,
		uint8(1),
		conditionRegisterData,
		uint8(1),
		[]byte{},
		defaultGenesisAccountAddress,
		big.NewInt(0),
		uint64(1))
	if err != nil {
		panic(err)
	}
	contractStorageMap[types.AddressConsensusGroup.String()][hex.EncodeToString(abi.GetConsensusGroupKey(types.SNAPSHOT_GID))] = hex.EncodeToString(snapshotConsensusGroupData)

	commonConsensusGroupData, err := abi.ABIConsensusGroup.PackVariable(abi.VariableNameConsensusGroupInfo,
		uint8(25),
		int64(3),
		int64(1),
		uint8(2),
		uint8(50),
		uint16(48),
		uint8(1),
		ledger.ViteTokenId,
		uint8(1),
		conditionRegisterData,
		uint8(1),
		[]byte{},
		defaultGenesisAccountAddress,
		big.NewInt(0),
		uint64(1))
	if err != nil {
		panic(err)
	}
	contractStorageMap[types.AddressConsensusGroup.String()][hex.EncodeToString(abi.GetConsensusGroupKey(types.DELEGATE_GID))] = hex.EncodeToString(commonConsensusGroupData)

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
	for index, addrStr := range addrStrList {
		nodeName := "s" + strconv.Itoa(index)
		addr, _ := types.HexToAddress(addrStr)
		registerData, err := abi.ABIConsensusGroup.PackVariable(abi.VariableNameRegistration, nodeName, addr, addr, helper.Big0, uint64(1), int64(1), int64(0), []types.Address{addr})
		if err != nil {
			panic(err)
		}
		contractStorageMap[types.AddressConsensusGroup.String()][hex.EncodeToString(abi.GetRegisterKey(nodeName, types.SNAPSHOT_GID))] = hex.EncodeToString(registerData)
		hisNameData, _ := abi.ABIConsensusGroup.PackVariable(abi.VariableNameHisName, nodeName)
		contractStorageMap[types.AddressConsensusGroup.String()][hex.EncodeToString(abi.GetHisNameKey(addr, types.SNAPSHOT_GID))] = hex.EncodeToString(hisNameData)
	}

	contractStorageMap[types.AddressMintage.String()] = make(map[string]string)
	viteTotalSupply := new(big.Int).Mul(big.NewInt(1e9), big.NewInt(1e18))
	mintageData, _ := abi.ABIMintage.PackVariable(abi.VariableNameTokenInfo,
		"Vite Token", "VITE", viteTotalSupply, uint8(18), defaultGenesisAccountAddress, big.NewInt(0), uint64(0), defaultGenesisAccountAddress, true, helper.Tt256m1, false)
	contractStorageMap[types.AddressMintage.String()][hex.EncodeToString(abi.GetMintageKey(ledger.ViteTokenId))] = hex.EncodeToString(mintageData)

	contractLogsMap := make(map[string][]config.GenesisVmLog)
	topics, data, err := abi.ABIMintage.PackEvent(abi.EventNameMint, ledger.ViteTokenId)
	if err != nil {
		panic(err)
	}
	contractLogsMap[types.AddressMintage.String()] = []config.GenesisVmLog{{Data: hex.EncodeToString(data), Topics: topics}}

	accountBalanceMap := make(map[string]map[string]*big.Int, 1)
	accountBalanceMap[defaultGenesisAccountAddress.String()] = map[string]*big.Int{ledger.ViteTokenId.String(): viteTotalSupply}

	return &config.Genesis{
		GenesisAccountAddress: &defaultGenesisAccountAddress,
		ContractStorageMap:    contractStorageMap,
		ContractLogsMap:       contractLogsMap,
		AccountBalanceMap:     accountBalanceMap,
	}
}
