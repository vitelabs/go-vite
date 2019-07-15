package ledger

import "github.com/vitelabs/go-vite/common/types"

type ContractMeta struct {
	Gid types.Gid // belong to the consensus group id

	SendConfirmedTimes uint8

	CreateBlockHash types.Hash // hash of send create block for creating the contract
	QuotaRatio      uint8      // the ratio of quota cost for the send block

	SeedConfirmedTimes uint8
}

const LengthBeforeSeedFork = types.GidSize + 1 + types.HashSize + 1

func (cm *ContractMeta) Serialize() []byte {
	buf := make([]byte, 0, LengthBeforeSeedFork+1)

	buf = append(buf, cm.Gid.Bytes()...)
	buf = append(buf, cm.SendConfirmedTimes)
	buf = append(buf, cm.CreateBlockHash.Bytes()...)
	buf = append(buf, cm.QuotaRatio)

	buf = append(buf, cm.SeedConfirmedTimes)

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

	if len(buf) <= LengthBeforeSeedFork {
		cm.SeedConfirmedTimes = cm.SendConfirmedTimes
		return nil
	}

	cm.SeedConfirmedTimes = buf[LengthBeforeSeedFork]
	return nil
}

func GetBuiltinContractMeta(addr types.Address) *ContractMeta {
	if types.IsBuiltinContractAddrInUseWithSendConfirm(addr) {
		return &ContractMeta{types.DELEGATE_GID, 1, types.Hash{}, getBuiltinContractQuotaRatio(addr), 0}
	} else if types.IsBuiltinContractAddrInUse(addr) {
		return &ContractMeta{types.DELEGATE_GID, 0, types.Hash{}, getBuiltinContractQuotaRatio(addr), 0}
	}
	return nil
}
func getBuiltinContractQuotaRatio(addr types.Address) uint8 {
	// TODO use special quota ratio for dex contracts
	return 10
}
