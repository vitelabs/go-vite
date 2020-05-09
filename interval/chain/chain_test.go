package chain

import (
	"testing"

	"fmt"

	"time"

	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/utils"
)

func TestGenesis(t *testing.T) {

	var genesisAccounts = []common.Address{common.Address("viteshan"), common.Address("jie")}
	for _, a := range genesisAccounts {
		genesis := common.NewAccountBlock(common.FirstHeight, "", "", a, time.Unix(0, 0),
			200, 0, common.GENESIS, &a, &a, nil)
		genesis.SetHash(utils.CalculateAccountHash(genesis))

		fmt.Println(a, &common.HashHeight{Hash: genesis.Hash(), Height: genesis.Height()})
	}

	var genesisAcc = []*common.AccountHashH{
		{common.HashHeight{Hash: "ccf131dac37a3ec9328290a9ad39c160baee02596daf303ad87d93815fce0a5a", Height: 0}, "viteshan"},
		{common.HashHeight{Hash: "904196e430c52d0687064a1723fa5124da7708e7e82d75924a846c4e84ac49c3", Height: 0}, "jie"},
	}

	var genesisSnapshot = common.NewSnapshotBlock(0, "a601ad0af8123a9dd85a201273276a82e41d6cc1e708bd62ea432dea76038639", "", "viteshan", time.Unix(1533550878, 0), genesisAcc)
	fmt.Println(utils.CalculateSnapshotHash(genesisSnapshot))
}

func TestInsert(t *testing.T) {
}
