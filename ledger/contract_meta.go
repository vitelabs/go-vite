package ledger

import "github.com/vitelabs/go-vite/common/types"

type ContractMeta struct {
	Gid                types.Gid
	SendConfirmedTimes uint8

	CreateBlockHash types.Hash
	QuotaRatio      uint8
}

func (cm *ContractMeta) Serialize() []byte {
	buf := make([]byte, 0, types.GidSize+1+types.HashSize+1)
	buf = append(buf, cm.Gid.Bytes()...)
	buf = append(buf, cm.SendConfirmedTimes)
	buf = append(buf, cm.CreateBlockHash.Bytes()...)
	buf = append(buf, cm.QuotaRatio)

	return buf
}

func (cm *ContractMeta) Deserialize(buf []byte) error {

	gid, err := types.BytesToGid(buf[:types.GidSize])
	if err != nil {
		return err
	}

	CreateBlockHashBuf := buf[1+types.GidSize : 1+types.GidSize+types.HashSize]
	CreateBlockHash, err := types.BytesToHash(CreateBlockHashBuf)
	if err != nil {
		return err
	}

	cm.Gid = gid
	cm.SendConfirmedTimes = buf[types.GidSize]
	cm.CreateBlockHash = CreateBlockHash
	cm.QuotaRatio = buf[types.GidSize+1+types.HashSize]

	return nil
}

func GetBuiltinContractMeta(addr types.Address) *ContractMeta {
	if types.IsBuiltinContractAddrInUseWithSendConfirm(addr) {
		return &ContractMeta{types.DELEGATE_GID, 1, types.Hash{}, 10}
	} else if types.IsBuiltinContractAddrInUse(addr) {
		return &ContractMeta{types.DELEGATE_GID, 0, types.Hash{}, 10}
	}
	return nil
}
