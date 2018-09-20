package contracts

import (
	"github.com/vitelabs/go-vite/abi"
	"strings"
	"testing"
)

func TestContractsABIInit(t *testing.T) {
	tests := []string{json_register, json_vote, json_pledge, json_consensusGroup, json_mintage}
	for _, data := range tests {
		if _, err := abi.JSONToABIContract(strings.NewReader(json_register)); err != nil {
			t.Fatalf("json to abi failed, %v, %v", data, err)
		}
	}
}
