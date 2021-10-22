package abi

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
}

func TestABI_MethodNameDexFundEndorseVxV2(t *testing.T) {
	result := "Kcu6jg=="
	byts, err := ABIDexFund.PackMethod(MethodNameDexFundEndorseVxV2)
	require.NoError(t, err)
	t.Log(base64.RawStdEncoding.EncodeToString(byts))
	assert.Equal(t, base64.StdEncoding.EncodeToString(byts), result)
}
