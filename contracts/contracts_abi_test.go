package contracts

import (
	"github.com/vitelabs/go-vite/abi"
	"strings"
	"testing"
)

func TestContractsABIInit(t *testing.T) {
	tests := []string{jsonRegister, jsonVote, jsonPledge, jsonConsensusGroup, jsonMintage}
	for _, data := range tests {
		if _, err := abi.JSONToABIContract(strings.NewReader(jsonRegister)); err != nil {
			t.Fatalf("json to abi failed, %v, %v", data, err)
		}
	}
}
