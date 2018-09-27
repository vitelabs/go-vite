package chain

import (
	"fmt"
	"testing"
)

func TestCreateAccount(t *testing.T) {
	//for i := 0; i < 25; i++ {
	//	addr, pri, _ := types.CreateAddress()
	//	fmt.Println(i + 1)
	//	fmt.Println(addr.String())
	//	fmt.Println(pri.Hex())
	//}

	fmt.Printf("%+v\n", GenesisMintageBlock)
	fmt.Printf("%+v\n", GenesisMintageBlockVC)

	fmt.Printf("%+v\n", GenesisMintageSendBlock)
	fmt.Printf("%+v\n", GenesisMintageSendBlockVC)

	fmt.Printf("%+v\n", GenesisConsensusGroupBlock)
	fmt.Printf("%+v\n", GenesisConsensusGroupBlockVC)

	fmt.Printf("%+v\n", GenesisRegisterBlock)
	fmt.Printf("%+v\n", GenesisRegisterBlockVC)

	fmt.Printf("%+v\n", GenesisSnapshotBlock)

}
