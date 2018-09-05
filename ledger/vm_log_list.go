package ledger

import "github.com/vitelabs/go-vite/common/types"

type VmLog struct {
	Topics []*types.Hash
	Data   []byte
}

type VmLogList []*VmLog

func (VmLogList) Serialize() ([]byte, error) {
	return nil, nil
}

func (VmLogList) DeSerialize([]byte) error {
	return nil
}
