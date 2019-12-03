package types

type QuotaInfo struct {
	BlockCount     uint64
	QuotaTotal     uint64
	QuotaUsedTotal uint64
}

type Quota struct {
	current                    uint64
	stakeQuotaPerSnapshotBlock uint64
	avg                        uint64
	snapshotCurrent            uint64
	blocked                    bool
	blockReleaseHeight         uint64
}

func NewQuota(stakeQuota, current, avg, snapshotCurrent uint64, blocked bool, blockReleaseHeight uint64) Quota {
	return Quota{current, stakeQuota, avg, snapshotCurrent, blocked, blockReleaseHeight}
}

func (q *Quota) StakeQuotaPerSnapshotBlock() uint64 {
	return q.stakeQuotaPerSnapshotBlock
}

// Current quota of a single account
func (q *Quota) Current() uint64 {
	return q.current
}

// Available quota in current snapshot block, excluding unconfirmed blocks
func (q *Quota) SnapshotCurrent() uint64 {
	return q.snapshotCurrent
}

func (q *Quota) Avg() uint64 {
	return q.avg
}

func (q *Quota) Blocked() bool {
	return q.blocked
}

func (q *Quota) BlockReleaseHeight() uint64 {
	return q.blockReleaseHeight
}
