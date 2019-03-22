package ledger

import "github.com/vitelabs/go-vite/common/types"

type ContractMeta struct {
	Gid                *types.Gid
	SendConfirmedTimes uint8
}

func (cm *ContractMeta) Serialize() []byte {
	return append(cm.Gid.Bytes(), cm.SendConfirmedTimes)
}

func (cm *ContractMeta) Deserialize(buf []byte) error {

	gid, err := types.BytesToGid(buf[:types.GidSize])
	if err != nil {
		return err
	}
	cm.Gid = &gid
	cm.SendConfirmedTimes = buf[types.GidSize:][0]
	return nil
}
