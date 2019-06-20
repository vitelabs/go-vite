package node

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestChainConfig(t *testing.T) {

	// JsonString

	//	VmLogWhiteList []types.Address `json:"vmLogWhiteList"` // contract address white list which save VM logs
	//	VmLogAll       *bool           `json:"vmLogAll"`       // save all VM logs, it will cost more disk space
	configJsonA := `{"VmLogWhiteList":["vite_d789431f1d820506c83fd539a0ae9863d6961382f67341a8b5"]}`

	configJsonB := `{"VmLogWhiteList":["vite_d789431f1d820506c83fd539a0ae9863d6961382f67341a8b5"],"VmLogAll":true}`

	config := &Config{}

	if err := json.Unmarshal([]byte (configJsonA), config); err != nil {
		t.Fatal(err)
	}

	configchain := config.makeChainConfig();
	if len(configchain.VmLogWhiteList) != 1 {
		t.Fatal("length must be 1")
	}
	if (configchain.VmLogWhiteList[0].String() != "vite_d789431f1d820506c83fd539a0ae9863d6961382f67341a8b5") {
		t.Fatal(" must be vite_d789431f1d820506c83fd539a0ae9863d6961382f67341a8b5")
	}

	fmt.Printf("vite %+v", configchain)

	configB := &Config{}

	if err := json.Unmarshal([]byte (configJsonB), configB); err != nil {
		t.Fatal(err)
	}

	configchainB := configB.makeChainConfig();

	if (configchainB.VmLogAll != true) {
		t.Fatal("VmLogAll must be true ")
	}

}
