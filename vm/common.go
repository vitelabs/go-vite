package vm

import (
	"bytes"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

var (
	DataResultPrefixSuccess = []byte{0}
	DataResultPrefixRevert  = []byte{1}
	DataResultPrefixFail    = []byte{2}

	Retry   = true
	NoRetry = false
)

func IsViteToken(tokenId types.TokenTypeId) bool {
	return bytes.Equal(tokenId.Bytes(), ledger.ViteTokenId().Bytes())
}
func IsSnapshotGid(gid types.Gid) bool {
	return bytes.Equal(gid.Bytes(), ledger.CommonGid().Bytes())
}

func useQuota(quota, cost uint64) (uint64, error) {
	if quota < cost {
		return 0, ErrOutOfQuota
	}
	quota = quota - cost
	return quota, nil
}
func useQuotaForData(data []byte, quota uint64) (uint64, error) {
	cost, err := dataGasCost(data)
	if err != nil {
		return 0, err
	}
	return useQuota(quota, cost)
}
