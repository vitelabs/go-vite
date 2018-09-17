package ledger

import "github.com/vitelabs/go-vite/common/types"

type VmLog struct {
	Topics []types.Hash
	Data   []byte
}

type VmLogList []*VmLog

func (VmLogList) Hash() *types.Hash {
	return nil
}

func (VmLogList) Serialize() ([]byte, error) {
	return nil, nil
}

func (VmLogList) Deserialize([]byte) error {
	return nil
}
