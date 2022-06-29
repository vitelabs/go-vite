package client

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/ledger/consensus/core"
	"github.com/vitelabs/go-vite/v2/rpcapi/api"
)

var RawUrl = "http://127.0.0.1:23456"
//var RawUrl = "http://127.0.0.1:48133"
//var RawUrl = "http://118.25.182.202:48132"
//var RawUrl = "http://139.199.74.104:48132"
//var RawUrl = "http://134.175.105.236:48132"

func TestGetBlockByHash(t *testing.T) {
	rpc := PreTestRpc(t, RawUrl)
	
	hash, err := types.HexToHash("bfff83c40823c60ff8b28430f988334e60f49a9adacfc4b94b2fce224aa97d14")
	if err != nil {
		t.Error(err)
		return
	}
	block, e := rpc.GetBlockByHash(hash)
	if e != nil {
		t.Error(e)
		return
	}
	t.Log(block)
	t.Log(block.TokenId)
}

func TestGetOnroadBlocksByAddress(t *testing.T) {
	rpc := PreTestRpc(t, RawUrl)

	addr, err := types.HexToAddress("vite_c4a8fe0c93156fe3fd5dc965cc5aea3fcb46f5a0777f9d1304")
	if err != nil {
		t.Error(addr)
		return
	}
	bs, e := rpc.GetOnroadBlocksByAddress(addr, 1, 10)
	if e != nil {
		t.Error(e)
		return
	}
	if len(bs) > 0 {
		for _, v := range bs {
			t.Log(v)
		}
	}
}

func TestCalcPoWDifficulty(t *testing.T) {
	t.Skip("Skipped by default. This test can be used to calculate PoW difficulty.")

	rpc := PreTestRpc(t, RawUrl)

	self, err := types.HexToAddress("vite_165a295e214421ef1276e79990533953e901291d29b2d4851f")
	if err != nil {
		t.Error(err)
		return
	}

	to, err := types.HexToAddress("vite_228f578d58842437fb52104b25750aa84a6f8558b6d9e970b1")
	if err != nil {
		t.Error(err)
		return
	}

	prevHash, err := types.HexToHash("58cb3cd2d00c6c0c883ec3aee9069445b826a165eacc75ece9e1fd008f6ccc5e")
	if err != nil {
		t.Fatal(err)
	}

	bs, e := rpc.CalcPoWDifficulty(api.CalcPoWDifficultyParam{
		SelfAddr:      self,
		PrevHash:      prevHash,
		BlockType:     ledger.BlockTypeSendCall,
		ToAddr:        &to,
		Data:          []byte("hello world"),
		UseStakeQuota: false,
	})

	if e != nil {
		t.Error(e)
		return
	}
	t.Log(bs)
}

func TestQueryReward(t *testing.T) {
	client := PreTestRpc(t, RawUrl)

	bs, e := client.GetRewardByIndex(0)

	if e != nil {
		t.Error(e)
		return
	}
	byt, _ := json.Marshal(bs)
	t.Log(string(byt))
}

func TestQueryVoteDetails(t *testing.T) {
	rpc := PreTestRpc(t, RawUrl)

	bs, e := rpc.GetVoteDetailsByIndex(0)

	if e != nil {
		t.Error(e)
		return
	}
	byt, _ := json.Marshal(bs)
	t.Log(string(byt))
}

func Test_GetConfirmedBalances(t *testing.T) {
	rpc := PreTestRpc(t, RawUrl)

	shash := types.HexToHashPanic("25e11b16de62fe5863266cac3c318cf603759a647049672fdb9db5524cc26282")
	gids := []types.TokenTypeId{ledger.ViteTokenId}

	data := []string{
		"vite_002f27f64a3e52b8ff62b28c4bb52441cb7d7dcf038032a52f",
		"vite_0033e7c54bd8bc63a4885aa194c15cb6d20465dc035cdab3a2",
		"vite_0065513a57258a84af95a438cf04efbb2071734cf29dabd7df",
		"vite_01b0cb6e49a9e1a86b76562a46f406efcf0ed14d31f9cc68a8",
		"vite_01c92aba4b6e5278e9c4b9fdd559bc9fe7ead97b30a2f55de5",
		"vite_02473e87c77ab8891dda88797764f960379c81b2380b749959",
		"vite_ffe984e5754cfcb852920147fcd931832d85f051363f50aee4",
	}

	var addrList []types.Address

	for _, v := range data {
		addrList = append(addrList, types.HexToAddressPanic(v))
	}

	balancesRes, err := rpc.GetConfirmedBalances(shash, addrList, gids)
	if err != nil {
		panic(err)
	}
	total := big.NewInt(0)
	for k, v := range balancesRes {
		for kk, vv := range v {
			fmt.Println(k, kk, vv.String())
			total.Add(total, vv)
		}
	}

	fmt.Println("total", total)

}

func TestSBPStats(t *testing.T) {
	rpc := PreTestRpc(t, RawUrl)

	stats, err := rpc.GetHourSBPStats(1, 0)
	if err != nil {
		t.Fatal(err)
	}

	rate := make(map[string][]*core.SbpStats)

	for _, v := range stats {
		//fmt.Println(k, v)
		stats := v["stat"]
		bytes, err := json.Marshal(stats)
		if err != nil {
			t.Fatal(err)
		}
		hourStats := &core.HourStats{}
		err = json.Unmarshal(bytes, hourStats)
		if err != nil {
			t.Fatal(err)
		}

		//fmt.Println(hourStats.Stats)
		totalNum := uint64(0)
		totalExcepted := uint64(0)
		for kk, vv := range hourStats.Stats {
			totalNum += vv.BlockNum
			totalExcepted += vv.ExceptedBlockNum
			rate[kk.String()] = append(rate[kk.String()], vv)
		}
		fmt.Println(v["stime"], totalNum, totalExcepted, float64(totalNum)/float64(totalExcepted))
	}

	for k, v := range rate {
		fmt.Print(k)
		for _, vv := range v {
			fmt.Printf("\t\t%.4f", float64(vv.BlockNum)/float64(vv.ExceptedBlockNum))
		}
		fmt.Println()

	}

	for k, v := range rate {
		fmt.Print(k)
		for _, vv := range v {
			fmt.Printf("\t\t(%d/%d)", vv.BlockNum, vv.ExceptedBlockNum)
		}
		fmt.Println()
	}

	for k, v := range rate {
		fmt.Print(k)
		for _, vv := range v {
			fmt.Printf("\t%d-%d", vv.ExceptedBlockNum-vv.BlockNum, vv.ExceptedBlockNum)
		}
		fmt.Println()
	}

	for k, v := range rate {
		fmt.Print(k)
		for _, vv := range v {
			fmt.Printf("\t%d", vv.ExceptedBlockNum-vv.BlockNum)
		}
		fmt.Println()
	}
}

func TestSbpHash(t *testing.T) {
	rpc := PreTestRpc(t, RawUrl)

	hashs := []string{
		"f348100aa8ef02f3dfa0938bc1c050073ddb2d73259357d3cbcb0610374350fc",
		"b111255964406a4c319fd41c941b6da7921273dea3139373bb7ce686623d6022",
		"ff044e6dff2fa64afa7d453d0addc663f93b560266759af42f69130b978687e4",
		"f2f071e4c09664d6023d9f5063e13c975e2d45249a15fc4d4e2521ec91b2ee0e",
		"b282eec7feaad79eff119637c5a5585a8d0ea468b8b3d4bb26f6d21ff4fded07",
		"bc6b714c5156467c771fc8e5faf933e8da67c4988b5846a9ed6f5d805e5a2e57",
		"cdd1d81a8cee589217f301b1acc4a571384340325c7e9df9aa673b2694406b2a",
		"c69280cc3daf4be24187fe3132046efa9dd4c4eba5e264dba55d2e4090b635c9",
	}

	for _, v := range hashs {
		block, err := rpc.GetSnapshotBlockByHash(types.HexToHashPanic(v))
		if err != nil {
			panic(err)
		}
		fmt.Println(v, block.Producer)
	}
}

func TestSbpddd(t *testing.T) {
	rpc := PreTestRpc(t, RawUrl)

	hash1 := types.HexToHashPanic("108f714be8fa1662eed21891d74694eae839408d9e28e19670be52dd177818a5")
	hash2 := types.HexToHashPanic("da5eb52fe8e39ca52698c47e7c6384b7a6c08177d44fc828f8dc562bf3157ad2")

	block, err := rpc.GetBlockByHash(hash1)
	if err != nil {
		panic(err)
	}
	block2, err := rpc.GetBlockByHash(hash2)
	if err != nil {
		panic(err)
	}
	t.Log(block.Data)
	t.Log(block2.Data)
}
