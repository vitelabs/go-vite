package chain_state

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"testing"
)

func TestStateDB_Write(t *testing.T) {

	//configJsonA := `{"VmLogWhiteList":["vite_d789431f1d820506c83fd539a0ae9863d6961382f67341a8b5"]}`
	//
	//
	//config := &Config{}
	//
	//if err := json.Unmarshal([]byte (configJsonA), config); err != nil {
	//	t.Fatal(err)
	//}
	//
	//configchain := config.makeChainConfig();
	//if len(configchain.VmLogWhiteList) != 1 {
	//	t.Fatal("length must be 1")
	//}
	//if (configchain.VmLogWhiteList[0].String() != "vite_d789431f1d820506c83fd539a0ae9863d6961382f67341a8b5") {
	//	t.Fatal(" must be vite_d789431f1d820506c83fd539a0ae9863d6961382f67341a8b5")
	//}

	address, err := types.HexToAddress("vite_d789431f1d820506c83fd539a0ae9863d6961382f67341a8b5")

	if err != nil {
		t.Fatal(fmt.Sprintf("address vite_d789431f1d820506c83fd539a0ae9863d6961382f67341a8b5 is not illegal. Error: %s", err.Error()));
	}
	stateDb, err := NewStateDB(nil, &config.Chain{
		VmLogWhiteList: []types.Address{address},
	}, "");
}
