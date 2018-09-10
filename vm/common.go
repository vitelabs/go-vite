package vm

var (
	DataResultPrefixSuccess = []byte{0}
	DataResultPrefixRevert  = []byte{1}
	DataResultPrefixFail    = []byte{2}
)

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
