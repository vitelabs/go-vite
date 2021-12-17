package abi

import (
	"encoding/base64"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vitelabs/go-vite/v2/common/types"
)

func TestABI_MethodNameDexFundPeriodJob(t *testing.T) {
	result := []string{
		"803SLgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAE=",
		"803SLgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAI=",
		"803SLgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAM=",
		"803SLgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQ=",
		"803SLgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAU=",
		"803SLgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAY=",
		"803SLgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAc=",
	}
	jobIds := []uint8{1, 2, 3, 4, 5, 6, 7}
	for i, jobId := range jobIds {
		lastPeriod := uint64(0)
		byts, err := ABIDexFund.PackMethod(MethodNameDexFundPeriodJob, lastPeriod+1, jobId)
		require.NoError(t, err)
		// t.Log(jobId, base64.StdEncoding.EncodeToString(byts))
		assert.Equal(t, base64.StdEncoding.EncodeToString(byts), result[i])
	}

	// jobIds = []uint8{3}
	// for peridId := 900; peridId <= 900; peridId++ {
	// 	for _, jobId := range jobIds {
	// 		lastPeriod := uint64(peridId)
	// 		byts, err := ABIDexFund.PackMethod(MethodNameDexFundPeriodJob, lastPeriod+1, jobId)
	// 		require.NoError(t, err)
	// 		// t.Log(jobId, base64.StdEncoding.EncodeToString(byts))

	// 		fmt.Printf("\"%s\",\n", base64.StdEncoding.EncodeToString(byts))
	// 		// assert.Equal(t, base64.StdEncoding.EncodeToString(byts), result[i])
	// 	}
	// }
}

func TestABI_MethodNameDexFundEndorseVxV2(t *testing.T) {
	result := "Kcu6jg=="
	byts, err := ABIDexFund.PackMethod(MethodNameDexFundEndorseVxV2)
	require.NoError(t, err)
	t.Log(base64.RawStdEncoding.EncodeToString(byts))
	assert.Equal(t, base64.StdEncoding.EncodeToString(byts), result)
}

func TestABI_MethodNameDexFundTradeAdminConfig(t *testing.T) {
	vtt, _ := types.HexToTokenTypeId("tti_2736f320d7ed1c2871af1d9d")
	btc, _ := types.HexToTokenTypeId("tti_322862b3f8edae3b02b110b1")
	eth, _ := types.HexToTokenTypeId("tti_06822f8d096ecdf9356b666c")
	usdt, _ := types.HexToTokenTypeId("tti_973afc9ffd18c4679de42e93")
	vite, _ := types.HexToTokenTypeId("tti_5649544520544f4b454e6e40")
	enable := true
	{
		data, err := ABIDexFund.PackMethod(MethodNameDexFundTradeAdminConfig, uint8(1), vtt, btc, enable, vite, uint8(1), uint8(1), big.NewInt(0), uint8(1), big.NewInt(0))
		require.NoError(t, err)
		t.Log(base64.RawStdEncoding.EncodeToString(data))
		result := "Qa/ldgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAACc28yDX7Rwoca8AAAAAAAAAAAAAAAAAAAAAAAAAAAAAMihis/jtrjsCsQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAFZJVEUgVE9LRU4AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
		assert.Equal(t, base64.StdEncoding.EncodeToString(data), result)
	}
	{
		data, err := ABIDexFund.PackMethod(MethodNameDexFundTradeAdminConfig, uint8(1), vtt, usdt, enable, vite, uint8(1), uint8(1), big.NewInt(0), uint8(1), big.NewInt(0))
		require.NoError(t, err)
		t.Log(base64.RawStdEncoding.EncodeToString(data))
		result := "Qa/ldgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAACc28yDX7Rwoca8AAAAAAAAAAAAAAAAAAAAAAAAAAAAAlzr8n/0YxGed5AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAFZJVEUgVE9LRU4AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
		assert.Equal(t, base64.StdEncoding.EncodeToString(data), result)
	}
	{
		data, err := ABIDexFund.PackMethod(MethodNameDexFundTradeAdminConfig, uint8(1), vtt, eth, enable, vite, uint8(1), uint8(1), big.NewInt(0), uint8(1), big.NewInt(0))
		require.NoError(t, err)
		t.Log(base64.RawStdEncoding.EncodeToString(data))
		result := "Qa/ldgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAACc28yDX7Rwoca8AAAAAAAAAAAAAAAAAAAAAAAAAAAAABoIvjQluzfk1awAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAFZJVEUgVE9LRU4AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
		assert.Equal(t, base64.StdEncoding.EncodeToString(data), result)
	}
	{
		data, err := ABIDexFund.PackMethod(MethodNameDexFundTradeAdminConfig, uint8(1), vtt, vite, enable, vite, uint8(1), uint8(1), big.NewInt(0), uint8(1), big.NewInt(0))
		require.NoError(t, err)
		t.Log(base64.RawStdEncoding.EncodeToString(data))
		result := "Qa/ldgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAACc28yDX7Rwoca8AAAAAAAAAAAAAAAAAAAAAAAAAAAAAVklURSBUT0tFTgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAFZJVEUgVE9LRU4AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
		assert.Equal(t, base64.StdEncoding.EncodeToString(data), result)
	}
}
